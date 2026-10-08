# So you want to get started with Agent Substrate and AX?

*A local lab for Kubernetes users, with Docker and Go.*

Running a coding agent in a Pod is straightforward. Deciding what to do with that Pod while the user is away is less obvious. You can leave it running, or stop it and rebuild the environment when the user comes back. A volume can keep the files. The process's memory is another matter.

That's what drew me to [Agent Substrate](https://github.com/agent-substrate/substrate/tree/7245baad8a6fd58f81c32338395b74484547fc12). Kubernetes runs a pool of worker Pods. Substrate runs applications, called *actors*, inside those workers. It can save an actor's state when you suspend it and restore it onto an available worker later. Waking an actor doesn't necessarily involve scheduling a new Pod.

If you run lots of sessions, this lets you save the idle ones instead of keeping every process alive. Kubernetes still manages the worker capacity.

[AX](https://github.com/google/ax/tree/ac2332829f22360ff97b0ba34d94dd0dd782f17e) builds on Substrate. You give it a Task and a Workspace, and it handles the underlying actor. Its YAML and `apply`/`get`/`watch` commands will look familiar if you use Kubernetes.

When I first tried this, I kept checking the worker Pods. I could create an actor, send it a request, suspend it, and bring it back, but the Pods hadn't changed. Watching a counter keep its in-memory value through that cycle helped me understand what Substrate was doing.

We'll do that first, then try an AX task and check which files survive a resume. We'll run the commands ourselves; no LLM account or API key is needed.

## Before you start

Use a Linux x86-64 machine with Docker running and Go installed. The setup command supplies Kind, kubectl, Git, Make, curl, and jq in a Docker container, so you don't need to install those on your host. Leave Go's [automatic toolchain downloads](https://go.dev/doc/toolchain) enabled.

Allow about 10–20 minutes for the exercises, plus setup time. Setup took about eight minutes on my machine with fresh source and Go caches, but some Docker layers were already cached. A first install may take longer. You'll also need several gigabytes of disk space for the downloads and builds.

Both projects are still changing quickly. The [companion repo](https://github.com/erain/substrate-ax-firsthand) uses specific commits and includes a small AX patch to make those versions work together. Stick with those versions for this walkthrough. This is a development lab; I haven't validated macOS or ARM, and I wouldn't use this setup for production.

## 1. Set up the lab

Download and extract the [ZIP archive](https://github.com/erain/substrate-ax-firsthand/archive/refs/heads/main.zip). Open a terminal in the extracted directory and run:

```sh
cd substrate-ax-firsthand-main
go run -buildvcs=false ./cmd/lab setup
```

The `-buildvcs=false` flag avoids requiring Git on your host. You'll see six setup stages:

```text
[1/6] Provision the private development toolbox
[2/6] Fetch the pinned Substrate and AX source
[3/6] Create Kind and connect the local image registry
[4/6] Install Substrate, snapshot storage, and gVisor workers
[5/6] Build the pinned CLIs, AX server, runner, and counter images
[6/6] Install AX and tutorial fixtures; wait for readiness
Lab ready ...
```

The script creates a Kind cluster, a local image registry, and a container with the development tools. It installs Substrate, its workers and local snapshot storage, then AX. You don't need a cloud account.

Use a private development machine: the tool container has access to your Docker socket. The lab keeps its own kubeconfig in `.local/kubeconfig` and leaves your normal context alone. Setup won't overwrite an existing cluster named `firsthand`. Keep the extracted directory where it is; the tool container mounts it.

When setup prints `Lab ready`, check the cluster:

```sh
./scripts/kube get nodes
./scripts/kube get pods -n ate-system
./scripts/kube get pods -n firsthand-lab
./scripts/kube get pods -n firsthand-ax
```

You should see a Ready node, running Substrate services and workers, and the AX server and Redis. A storage-initialization Job showing `Completed` is fine. `scripts/kube` runs kubectl inside the tool container.

If setup fails, check `.local/setup.log` and the [troubleshooting guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/TROUBLESHOOTING.md) before retrying. Once the script has recorded successful cluster creation, rerunning setup resumes the install without recreating the cluster.

## 2. Try Substrate with a counter

Open a second terminal in the same directory and run:

```sh
./scripts/router
```

Leave it running. This forwards Substrate's router to `127.0.0.1:18080`; our requests will go through it. If that port is in use, the [README](https://github.com/erain/substrate-ax-firsthand#1-try-the-counter) explains how to pick another one.

Back in your first terminal, open `.local/rendered/counter-template.yaml`. Setup generated the full file for you. Here's the part we'll use:

```yaml
workerSelector:
  matchLabels:
    workload: firsthand-counter
# The full manifest mounts a durable directory at /data.
snapshotConfig:
  onPause: SNAPSHOT_CONTENT_SCOPE_FULL
  onCommit: SNAPSHOT_CONTENT_SCOPE_FULL
  storageLocation: gs://ate-snapshots/firsthand/counter/
```

The ActorTemplate describes our application: the counter image, its settings, and its durable `/data` directory. The WorkerPool describes the worker Pods, including their runtime image. These are two different images. The label selector tells Substrate which workers the counter can use.

Create a counter actor and record the worker Pod's UID and restart count:

```sh
./scripts/ate create actor counter-one -a firsthand --template counter
./scripts/ate get actor counter-one -a firsthand -o yaml
./scripts/kube get pods -n firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

The new actor should be suspended. We'll wake it by sending a request rather than calling resume.

The counter keeps two numbers: `memoryCount` in process memory, and `fileCount` in `/data/count.txt`. Every POST increments both. Give the request wrapper a shorter name and call it twice:

```sh
hit() {
  ./scripts/request
}
hit
hit
```

You should see:

```json
{"fileCount":1,"memoryCount":1}
{"fileCount":2,"memoryCount":2}
```

`scripts/request` runs curl in the tool container. It sends a POST through the router with the header `ate-target-actor: firsthand/counter-one`. The first request wakes the actor.

Now suspend it:

```sh
./scripts/ate suspend actor counter-one -a firsthand
./scripts/ate get actor counter-one -a firsthand -o yaml
```

In the YAML, look for `ACTOR_STATE_SUSPENDED` and `status.externalSnapshot`. The snapshot's content scope is `FULL`: it includes the actor's memory as well as filesystem state. The URI points to the saved snapshot. Although it starts with `gs://`, this lab uses local storage, not your own Google Cloud bucket.

What would you expect `memoryCount` to be if the next request started a new process? Keep that in mind and try another request. Don't resume the actor manually.

```sh
hit
./scripts/kube get pods -n firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

You should get `{"fileCount":3,"memoryCount":3}`. The worker Pod should still have the same UID and restart count.

The memory value matters here. The counter starts it at zero and never loads it from the saved file; you can check that in [its source](https://github.com/erain/substrate-ax-firsthand/blob/main/cmd/counter/main.go). If Substrate had started a new counter process with the old file, you'd see `fileCount:3` and `memoryCount:1`. Both are `3` because it restored the process state.

The worker Pod was already there. The request brought the actor back inside that capacity. That's the behavior I couldn't see just by listing Pods.

Suspend the counter again to free up worker capacity:

```sh
./scripts/ate suspend actor counter-one -a firsthand
```

## 3. Try an AX task

AX creates Substrate actors for you. Here we'll give it a Task with a Workspace mounted at `/workspace`.

Open `.local/rendered/ax-task.yaml`. It looks like this, with a real image digest filled in:

```yaml
apiVersion: ax.io/v1alpha1
kind: Workspace
metadata:
  name: scratch
  atespace: firsthand
spec: {}
---
apiVersion: ax.io/v1alpha1
kind: Task
metadata:
  name: task-one
  atespace: firsthand
spec:
  image: YOUR_GENERATED_RUNNER_IMAGE_DIGEST
  debug: true
  workspaces:
    - name: scratch
      path: /workspace
```

Use the generated file, not the example above: `YOUR_GENERATED_RUNNER_IMAGE_DIGEST` is a placeholder. Apply it with AX:

```sh
./scripts/ax apply -f .local/rendered/ax-task.yaml
./scripts/ax resume task task-one -a firsthand
```

This version of AX creates the task suspended, so we resume it explicitly. The task uses a debug runner with no `spec.command`; we'll run the calculation ourselves once it's ready.

Run this inside the task:

```sh
./scripts/ax ssh task-one -a firsthand -- sh -ec '
  printf "%s\n" 10 20 30 | awk "{total+=\$1} END {print total}" > /workspace/result.txt
  echo transient > /tmp/firsthand-marker
  cat /workspace/result.txt
'
```

You should see `60`. The command adds three numbers, saves the answer in the workspace, and creates a marker in `/tmp`. AX calls this command `ssh`, but it executes in the sandbox without an SSH daemon.

We now have two files in different places. Will both still be there after a suspend and resume?

```sh
./scripts/ax suspend task task-one -a firsthand
./scripts/ax resume task task-one -a firsthand
./scripts/ax ssh task-one -a firsthand -- sh -ec '
  cat /workspace/result.txt
  test ! -e /tmp/firsthand-marker
  echo "Transient marker is gone"
'
```

Expected:

```text
60
Transient marker is gone
```

The result is still in `/workspace`, but the marker is gone. AX's generated ActorTemplate uses `DATA` snapshots. It saves the durable workspace and restores it into a fresh runtime. The writable image layer, where we put the `/tmp` marker, doesn't survive.

This is different from the counter's `FULL` snapshot:

| Experiment | Snapshot | What survived |
|---|---|---|
| Substrate counter | FULL | RAM counter and durable file |
| AX task | DATA | Durable workspace result, not the `/tmp` marker |

You can inspect the Substrate actor and template that AX created. Then suspend the task when you're done:

```sh
./scripts/ate get actor task-one -a firsthand -o yaml
./scripts/ate get actor-templates -a firsthand
./scripts/ax suspend task task-one -a firsthand
```

A couple of details about this AX version: its generated template doesn't select the counter's worker pool or pass through Task resource limits. The task may use other eligible gVisor workers in the lab. Also, a task marked `Running` means the runner is alive, not that your calculation succeeded. Check the command's output and exit status for that.

## Clean up

Stop the router in your second terminal with Ctrl-C. Read the [cleanup guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/CLEANUP.md), then remove the tutorial resources:

```sh
CONFIRM_TUTORIAL_CLEANUP=yes ./bin/lab cleanup
```

This deletes the tutorial's actors, task, workspace, worker pool, and AX services. You'll lose access to the counter and result through those resources. It leaves the Kind cluster, shared Substrate services, other labs, and stored snapshots in place. Rerun the setup command if you want to do the walkthrough again.
