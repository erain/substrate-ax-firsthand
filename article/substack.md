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
[6/6] Install AX and tutorial namespaces; wait for readiness
Lab ready ...
```

The script creates a Kind cluster, a local image registry, and a container with the development tools. It installs Substrate, some initial workers and local snapshot storage, then AX. We'll create our counter's worker pool and template ourselves. You don't need a cloud account.

Use a private development machine: the tool container has access to your Docker socket. The lab keeps its own kubeconfig in `.local/kubeconfig` and leaves your normal context alone. Setup won't overwrite an existing cluster named `firsthand`. Keep the extracted directory where it is; the tool container mounts it.

When setup prints `Lab ready`, enter the tool container:

```sh
./bin/lab shell
```

You now have the real `kubectl`, `kubectl ate`, `ax`, and `curl` commands on your PATH. This shell uses the lab's private kubeconfig and AX tunnel cache. We'll name the context and namespaces explicitly in the commands below. Type `exit` whenever you want to return to your host.

Check the cluster from this shell:

```sh
kubectl --context kind-firsthand get nodes
kubectl --context kind-firsthand get pods --namespace ate-system
kubectl --context kind-firsthand get namespace firsthand-lab firsthand-ax
kubectl --context kind-firsthand get pods --namespace firsthand-ax
```

You should see a Ready node, running Substrate services, both tutorial namespaces, and the AX server and Redis. A storage-initialization Job showing `Completed` is fine. There are no counter Pods in `firsthand-lab` yet.

If setup fails, check `.local/setup.log` and the [troubleshooting guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/TROUBLESHOOTING.md) before retrying. Once the script has recorded successful cluster creation, rerunning setup resumes the install without recreating the cluster.

<a name="1-try-the-counter"></a>

## 2. Try Substrate with a counter

Open a second terminal in the same directory on your host and enter the tool container there too:

```sh
./bin/lab shell
```

In that shell, start the port-forward and leave it running:

```sh
kubectl --context kind-firsthand --namespace ate-system \
  port-forward service/atenet-router "${TUTORIAL_PORT}:80" \
  --address=127.0.0.1
```

This forwards the router Service's port 80 to a local port, normally `18080`. The shell supplies `TUTORIAL_PORT` from setup; it also works if you chose another port. If the port is occupied, exit both tool shells and rerun setup on your host with another port, for example `go run -buildvcs=false ./cmd/lab --port 28080 setup`. Reenter the shells to pick up the saved port. The [setup guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/SETUP.md#configuring-local-commands) has more details.

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

Apply the WorkerPool to Kubernetes. Its controller creates a Deployment, so wait for that to appear and become ready:

```sh
kubectl --context kind-firsthand apply -f .local/rendered/worker-pool.yaml
kubectl --context kind-firsthand --namespace firsthand-lab \
  wait --for=create deployment/firsthand-workers --timeout=60s
kubectl --context kind-firsthand --namespace firsthand-lab \
  rollout status deployment/firsthand-workers --timeout=180s
```

Now create the ActorTemplate through Substrate's API:

```sh
kubectl ate --context kind-firsthand \
  create actor-template -f .local/rendered/counter-template.yaml
kubectl ate --context kind-firsthand \
  get actor-template counter --atespace firsthand -o yaml
```

Wait until `status.goldenSnapshotStatus.goldenTag.name` has a value. If it's still preparing, rerun the get command; if it reports an error, check the [troubleshooting guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/TROUBLESHOOTING.md). Substrate prepares a starting snapshot for actors created from this template.

Notice the two APIs: `kubectl apply` submitted a Kubernetes CRD, while `kubectl ate` called Substrate. ActorTemplates aren't Kubernetes CRDs. Setup already created the `firsthand` atespace, which groups our Substrate actors; it's separate from the `firsthand-lab` Kubernetes namespace.

Create a counter actor and record the worker Pod's UID and restart count:

```sh
kubectl ate --context kind-firsthand \
  create actor counter-one --atespace firsthand --template counter
kubectl ate --context kind-firsthand \
  get actor counter-one --atespace firsthand -o yaml
kubectl --context kind-firsthand get pods --namespace firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

The new actor should be suspended. We'll wake it by sending a request rather than calling resume.

The counter keeps two numbers: `memoryCount` in process memory, and `fileCount` in `/data/count.txt`. Every POST increments both. Send two requests:

```sh
curl --fail --silent --show-error --max-time 60 \
  --request POST \
  --header 'ate-target-actor: firsthand/counter-one' \
  "http://127.0.0.1:${TUTORIAL_PORT}/"

curl --fail --silent --show-error --max-time 60 \
  --request POST \
  --header 'ate-target-actor: firsthand/counter-one' \
  "http://127.0.0.1:${TUTORIAL_PORT}/"
```

You should see:

```json
{"fileCount":1,"memoryCount":1}
{"fileCount":2,"memoryCount":2}
```

The header `ate-target-actor: firsthand/counter-one` tells the router which actor should receive the request. The first request wakes it.

Now suspend it:

```sh
kubectl ate --context kind-firsthand \
  suspend actor counter-one --atespace firsthand
kubectl ate --context kind-firsthand \
  get actor counter-one --atespace firsthand -o yaml
```

In the YAML, look for `ACTOR_STATE_SUSPENDED` and `status.externalSnapshot`. The snapshot's content scope is `FULL`: it includes the actor's memory as well as filesystem state. The URI points to the saved snapshot. Although it starts with `gs://`, this lab uses local storage, not your own Google Cloud bucket.

What would you expect `memoryCount` to be if the next request started a new process? Keep that in mind and try another request. Don't resume the actor manually.

```sh
curl --fail --silent --show-error --max-time 60 \
  --request POST \
  --header 'ate-target-actor: firsthand/counter-one' \
  "http://127.0.0.1:${TUTORIAL_PORT}/"
kubectl --context kind-firsthand get pods --namespace firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

You should get `{"fileCount":3,"memoryCount":3}`. The worker Pod should still have the same UID and restart count.

The memory value matters here. The counter starts it at zero and never loads it from the saved file; you can check that in [its source](https://github.com/erain/substrate-ax-firsthand/blob/main/cmd/counter/main.go). If Substrate had started a new counter process with the old file, you'd see `fileCount:3` and `memoryCount:1`. Both are `3` because it restored the process state.

The worker Pod was already there. The request brought the actor back inside that capacity. That's the behavior I couldn't see just by listing Pods.

Suspend the counter again to free up worker capacity:

```sh
kubectl ate --context kind-firsthand \
  suspend actor counter-one --atespace firsthand
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

Use the generated file, not the example above: `YOUR_GENERATED_RUNNER_IMAGE_DIGEST` is a placeholder. Apply it with AX, not `kubectl apply`:

```sh
ax --context kind-firsthand --namespace firsthand-ax \
  apply -f .local/rendered/ax-task.yaml
ax --context kind-firsthand --namespace firsthand-ax \
  resume task task-one --atespace firsthand
```

This version of AX creates the task suspended, so we resume it explicitly. The task uses a debug runner with no `spec.command`; we'll run the calculation ourselves once it's ready.

Run this inside the task:

```sh
ax --context kind-firsthand --namespace firsthand-ax \
  ssh task-one --atespace firsthand -- sh -ec '
  printf "%s\n" 10 20 30 | awk "{total+=\$1} END {print total}" > /workspace/result.txt
  echo transient > /tmp/firsthand-marker
  cat /workspace/result.txt
'
```

You should see `60`. The command adds three numbers, saves the answer in the workspace, and creates a marker in `/tmp`. AX calls this command `ssh`, but it executes in the sandbox without an SSH daemon.

We now have two files in different places. Will both still be there after a suspend and resume?

```sh
ax --context kind-firsthand --namespace firsthand-ax \
  suspend task task-one --atespace firsthand
ax --context kind-firsthand --namespace firsthand-ax \
  resume task task-one --atespace firsthand
ax --context kind-firsthand --namespace firsthand-ax \
  ssh task-one --atespace firsthand -- sh -ec '
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

The two experiments show what each snapshot saves:

- The counter's `FULL` snapshot keeps both the in-memory counter and the durable file.
- The AX task's `DATA` snapshot keeps the workspace result, but not the `/tmp` marker.

You can inspect the Substrate actor and template that AX created. Then suspend the task when you're done:

```sh
kubectl ate --context kind-firsthand \
  get actor task-one --atespace firsthand -o yaml
kubectl ate --context kind-firsthand \
  get actor-templates --atespace firsthand
ax --context kind-firsthand --namespace firsthand-ax \
  suspend task task-one --atespace firsthand
```

A couple of details about this AX version: its generated template doesn't select the counter's worker pool or pass through Task resource limits. The task may use other eligible gVisor workers in the lab. Also, a task marked `Running` means the runner is alive, not that your calculation succeeded. Check the command's output and exit status for that.

## Clean up

Stop the port-forward in your second terminal with Ctrl-C. Type `exit` in each tool shell to return to your host. Read the [cleanup guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/CLEANUP.md), then run this from the repository on your host:

```sh
CONFIRM_TUTORIAL_CLEANUP=yes ./bin/lab cleanup
```

This deletes the tutorial's actors, task, workspace, worker pool, and AX services. You'll lose access to the counter and result through those resources. It leaves the Kind cluster, shared Substrate services, other labs, and stored snapshots in place. Rerun the setup command if you want to do the walkthrough again.

For setup options and automated tests, see the [setup guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/SETUP.md). The [validation notes](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/VALIDATION.md) record what we've tested and what we haven't. Upstream attribution is in [NOTICE](https://github.com/erain/substrate-ax-firsthand/blob/main/NOTICE).
