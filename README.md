# So you want to get started with Agent Substrate and AX?

This repo sets up a local Kind cluster with Substrate and AX, then walks you through two examples. First, you'll suspend a counter and wake it with an HTTP request. Then you'll write a file in an AX workspace and see what survives a resume.

You need a Linux x86-64 machine, a running local Docker daemon, and modern Go with automatic toolchain downloads enabled. Setup supplies Kind, kubectl, Git, Make, curl, and jq in a Docker container. No model account or API key is needed.

Allow about 10–20 minutes for the exercises, plus the initial downloads and builds. This is a development lab, not a production setup. See [validation](docs/VALIDATION.md) for what we've tested and which platforms we haven't.

Follow the steps below, or read the [article draft](article/substack.md) for more background. Both projects are changing quickly, so use the versions in [versions.env](versions.env). The [setup guide](docs/SETUP.md#why-is-there-an-ax-patch) explains the small AX patch included here.

## 0. Set up the lab

Download and extract the [repository ZIP](https://github.com/erain/substrate-ax-firsthand/archive/refs/heads/main.zip), or clone [erain/substrate-ax-firsthand](https://github.com/erain/substrate-ax-firsthand) if you already use Git, then work from its root.

With a running local Docker daemon and modern Go (automatic toolchain downloads enabled), run:

```sh
go run -buildvcs=false ./cmd/lab setup
```

The launcher creates a tool container, downloads the pinned source, creates Kind and a local registry, and installs Substrate and AX. It builds the sample images and waits for the services and worker Pods to be ready. Detailed output goes to `.local/setup.log`. The tool container mounts your Docker socket, so use a trusted development machine.

When setup finishes, enter the tool container:

```sh
./bin/lab shell
```

This adds the built CLIs to PATH and selects the private kubeconfig and AX tunnel cache. It doesn't alias or replace the commands. Use the actual `kubectl`, `kubectl ate`, `ax`, and `curl` tools below. Type `exit` to return to your host.

Check the cluster from this shell:

```sh
kubectl --context kind-firsthand get nodes
kubectl --context kind-firsthand get pods --namespace ate-system
kubectl --context kind-firsthand get namespace firsthand-lab firsthand-ax
kubectl --context kind-firsthand get pods --namespace firsthand-ax
```

You should see a Ready node, healthy Substrate services, both tutorial namespaces, and the AX server and Redis running. A storage-initialization Job marked `Completed` is normal. We'll create the counter's worker pool, template, and actor below; `firsthand-lab` has no counter Pods yet.

The lab uses `.local/kubeconfig` and leaves your usual context alone. Setup refuses an existing cluster it doesn't recognize. If installation fails after successful cluster creation has been recorded, check the log and rerun setup; it resumes without recreating the cluster. See [SETUP.md](docs/SETUP.md) for details and the optional existing-lab route.

You'll find the generated YAML in `.local/rendered/`. Its image digests come from the local registry; nothing is pushed to a public registry. Keep this directory where it is, because the tool container mounts it.

The commands name `kind-firsthand` explicitly. For AX, also pass `--namespace firsthand-ax` to reach the tutorial server. The shell removes inherited `AX_SERVER` overrides and keeps AX's tunnels separate from other labs. [SETUP.md](docs/SETUP.md#configuring-local-commands) has the connection details.

## 1. Try the counter

Open a second terminal in this directory on your host and enter the tool container:

```sh
./bin/lab shell
```

In that shell, start the router forward:

```sh
kubectl --context kind-firsthand --namespace ate-system \
  port-forward service/atenet-router "${TUTORIAL_PORT}:80" \
  --address=127.0.0.1
```

Leave it running. The local port is `18080` unless you chose another during setup. If it's occupied, exit the tool shells and rerun setup on your host with another port, for example `go run -buildvcs=false ./cmd/lab --port 28080 setup`. Reenter the shells so they pick up the saved port. Leave unrelated port-forwards alone.

Back in your first terminal, look at the two manifests:

```sh
sed -n '1,100p' .local/rendered/worker-pool.yaml
sed -n '1,120p' .local/rendered/counter-template.yaml
```

The WorkerPool is a Kubernetes resource. It describes the worker Pods and their runtime image. The ActorTemplate belongs to Substrate's API: it describes the counter application image, which workers it can use, its durable `/data` directory, and its `FULL` snapshot settings. The application image and worker runtime image are different.

Apply the WorkerPool, wait for its controller to create a Deployment, then wait for the rollout:

```sh
kubectl --context kind-firsthand apply -f .local/rendered/worker-pool.yaml
kubectl --context kind-firsthand --namespace firsthand-lab \
  wait --for=create deployment/firsthand-workers --timeout=60s
kubectl --context kind-firsthand --namespace firsthand-lab \
  rollout status deployment/firsthand-workers --timeout=180s
```

Create the ActorTemplate with the Substrate plugin:

```sh
kubectl ate --context kind-firsthand \
  create actor-template -f .local/rendered/counter-template.yaml
kubectl ate --context kind-firsthand \
  get actor-template counter --atespace firsthand -o yaml
```

Wait until `status.goldenSnapshotStatus.goldenTag.name` has a value, rerunning the get command if necessary. Substrate is preparing the starting snapshot for new actors. Don't continue if the status reports an error; see [troubleshooting](docs/TROUBLESHOOTING.md#actortemplate-has-no-golden-tag).

The `kubectl ate` plugin talks to Substrate's API; ActorTemplates aren't Kubernetes CRDs. Setup created the `firsthand` atespace for our actors. That is separate from the Kubernetes namespace `firsthand-lab` that holds our worker Pods.

Create the actor and check its state:

```sh
kubectl ate --context kind-firsthand \
  create actor counter-one --atespace firsthand --template counter
kubectl ate --context kind-firsthand \
  get actor counter-one --atespace firsthand -o yaml
```

The actor should start suspended. Record the worker Pod's UID and restart count so we can compare them later:

```sh
kubectl --context kind-firsthand get pods --namespace firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

Our counter increments two numbers on each POST: one in memory and one in `/data/count.txt`. Send two requests:

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

Expected:

```json
{"fileCount":1,"memoryCount":1}
{"fileCount":2,"memoryCount":2}
```

The header `ate-target-actor: firsthand/counter-one` selects the actor. The first request wakes it. Check that it's running, then suspend it:

```sh
kubectl ate --context kind-firsthand \
  get actor counter-one --atespace firsthand -o yaml
kubectl ate --context kind-firsthand \
  suspend actor counter-one --atespace firsthand
kubectl ate --context kind-firsthand \
  get actor counter-one --atespace firsthand -o yaml
```

Look for `status.state: ACTOR_STATE_SUSPENDED` and `status.externalSnapshot`. The snapshot scope should be `FULL`, which includes memory and filesystem state. Its URI identifies the saved snapshot. This lab uses local storage for that `gs://` location; you don't need a Google Cloud account.

What would happen to the memory counter if this started a new process? Try another request without calling resume:

```sh
curl --fail --silent --show-error --max-time 60 \
  --request POST \
  --header 'ate-target-actor: firsthand/counter-one' \
  "http://127.0.0.1:${TUTORIAL_PORT}/"
kubectl --context kind-firsthand get pods --namespace firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

You should get `{"fileCount":3,"memoryCount":3}`, with the same worker Pod UID and restart count.

The counter never initializes its memory value from the saved file. A new process with that file would return `fileCount:3` and `memoryCount:1`; the unit test covers this case. Getting both values back shows that Substrate restored the process state. Checking the Pod tells us that it used the existing worker capacity. We haven't tested worker failures or exactly-once request handling here.

Suspend the counter again to free up capacity:

```sh
kubectl ate --context kind-firsthand \
  suspend actor counter-one --atespace firsthand
```

## 2. Try an AX task

AX creates the underlying Substrate actor for us. Inspect the manifest, apply it through AX, and resume the task:

```sh
sed -n '1,120p' .local/rendered/ax-task.yaml
ax --context kind-firsthand --namespace firsthand-ax \
  apply -f .local/rendered/ax-task.yaml
ax --context kind-firsthand --namespace firsthand-ax \
  get task task-one --atespace firsthand
ax --context kind-firsthand --namespace firsthand-ax \
  resume task task-one --atespace firsthand
```

The manifest has an empty Workspace named `scratch`, mounted at `/workspace`, and a Task using our debug runner. This AX version creates tasks suspended, so we resume explicitly. We leave `spec.command` empty and run the calculation ourselves once the task is ready.

Add three numbers and save the result in the workspace. Also create a marker in `/tmp`:

```sh
ax --context kind-firsthand --namespace firsthand-ax \
  ssh task-one --atespace firsthand -- sh -ec '
  printf "%s\n" 10 20 30 | awk "{total+=\$1} END {print total}" > /workspace/result.txt
  echo transient > /tmp/firsthand-marker
  cat /workspace/result.txt
'
```

You should see `60`. AX calls this command `ssh`, but it executes in the sandbox without an SSH daemon.

Will both files still be there after a resume? Suspend the task, resume it, and check:

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

AX's generated ActorTemplate uses `DATA` snapshots. It restores the durable workspace into a fresh runtime, so `result.txt` survives. The writable image layer containing the `/tmp` marker doesn't. This differs from the counter's `FULL` snapshot, which also restored memory.

Look at the actor and template AX created, then suspend the task when you're done:

```sh
kubectl ate --context kind-firsthand \
  get actor task-one --atespace firsthand -o yaml
kubectl ate --context kind-firsthand \
  get actor-templates --atespace firsthand
ax --context kind-firsthand --namespace firsthand-ax \
  suspend task task-one --atespace firsthand
```

This AX version doesn't set a dedicated worker selector or pass through Task resource limits in its generated template. The counter selects `firsthand-workers`, but AX can use other eligible gVisor workers in this lab. Also, `Running` means the task's runner is alive, not that your command succeeded. Use the command's output and exit status to check that.

## Where each resource fits

| API | Objects you used | What they manage |
|---|---|---|
| Kubernetes | WorkerPool / worker Pod | Worker capacity |
| Substrate | ActorTemplate / Actor | Application settings, requests, and saved execution state |
| AX | Workspace / Task | Task environment and the underlying Substrate actor |

AX also has a `Model` resource. We haven't used it here: the calculation runs through the debug command, not an autonomous agent or model call.

## Tests, recovery, and cleanup

Run these checks from your host terminal:

```sh
./bin/lab test      # unit tests, race detector, vet, shell syntax, safety guards
./bin/lab smoke     # end-to-end assertions; requires unused tutorial names
```

The smoke test uses the same actor, task, and workspace names as this walkthrough. If they already exist, it stops rather than overwriting them. Run it in a clean tutorial scope. See [validation](docs/VALIDATION.md) for test results and [troubleshooting](docs/TROUBLESHOOTING.md) if something doesn't match the expected output.

When finished with the walkthrough, suspend both workloads and stop the port-forward with Ctrl-C. Type `exit` in both tool shells. Read [CLEANUP.md](docs/CLEANUP.md), then remove the tutorial resources from your host:

```sh
CONFIRM_TUTORIAL_CLEANUP=yes ./bin/lab cleanup
```

Cleanup deletes the tutorial resources and their live state. You'll lose access to the counter and result through those resources. It leaves other labs, shared Substrate, the cluster, registry, and stored snapshots in place. Rerun setup to prepare another walkthrough on the same cluster.

See [NOTICE](NOTICE) for upstream attribution and [the article publishing checklist](article/PUBLISHING.md) before publishing the Substack draft.
