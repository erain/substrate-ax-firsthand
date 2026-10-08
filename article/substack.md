# So you want to get started with Agent Substrate and AX?

*Two hands-on experiments: pause a process that remembers, then resume a task whose workspace survives.*

If you know Kubernetes, you know how to run an agent in a Pod. But **should every agent session need its own live Pod—even when it has nothing to do?**

Imagine a platform hosting thousands of coding-agent sessions. Each has files, tools, and execution state. Some are working; others are waiting for a human to come back. In a one-Pod-per-session design, you can keep their processes alive to preserve that context, or stop them and arrange to reconstruct it later. A persistent volume helps with files; it doesn't, by itself, preserve process memory.

[Agent Substrate](https://github.com/agent-substrate/substrate/tree/7245baad8a6fd58f81c32338395b74484547fc12) explores a different split: Kubernetes manages a pool of ready worker Pods; Substrate manages the actors that use them. Actors can be suspended into snapshots and restored onto available workers. Activating an actor need not mean scheduling a new Pod. The design separates the number of sessions you retain from the execution capacity they currently occupy.

[AX](https://github.com/google/ax/tree/ac2332829f22360ff97b0ba34d94dd0dd782f17e) adds the developer-facing layer: declarative Tasks, Workspaces, and Model configuration, with an `apply`/`get`/`watch` workflow that Kubernetes users will recognize.

If you build developer platforms or operate sandboxed workloads, that's the reason to explore both: **how do you share compute without treating every pause as starting over—and what API should developers use to ask for it?** Your Kubernetes knowledge remains useful; the interesting part is deciding what belongs above the Pod layer. These are early-stage projects to investigate, not a recommendation to migrate production workloads.

I understood that split much better after watching a counter wake up, remember its in-memory value, and run inside the same worker Pod. That's what we'll do here, then contrast it with an AX task whose workspace survives but whose runtime starts fresh. No LLM key, no model bill, and no need to take a performance claim on faith.

**Start here:** a Linux/amd64 development machine with **Docker running and Go installed**. You don't need Kind, kubectl, Git, Make, an existing cluster, a cloud account, or an LLM key. We will install the lab here, then do the two experiments. Modern Go can download the required toolchain automatically; leave that default enabled. [Go toolchain documentation](https://go.dev/doc/toolchain).

Allow 10–20 minutes for the experiments, plus the initial downloads and builds. Setup is part of this walkthrough, not homework; its total time depends on your machine and network. This is an experimental local lab, not a production recipe. macOS and ARM are not validated by this edition.

For a reference point, my fresh-cluster setup took about eight minutes with fresh source/Go caches but some Docker layers already cached. A completely cold machine may take longer, and the setup/cache downloads use several gigabytes of disk.

**Companion code:** [substrate-ax-firsthand](https://github.com/erain/substrate-ax-firsthand). It pins both projects and includes the small AX API adjustment required for this combination. Please don't mix these samples with two moving `main` branches.

## The mental model, in three lines

Kubernetes supplies worker Pods: ready execution capacity.

[Agent Substrate](https://github.com/agent-substrate/substrate/tree/7245baad8a6fd58f81c32338395b74484547fc12) manages actors within that capacity: activation, request routing, snapshots, suspend, and restore.

[AX](https://github.com/google/ax/tree/ac2332829f22360ff97b0ba34d94dd0dd782f17e) adds developer-facing Tasks and Workspaces on top. A Model is another AX primitive, but we won't exercise it today.

The interesting difference is that creating or waking an actor does not inherently require creating a new Kubernetes Pod. A prepared worker can execute it. We will observe one lifecycle—not benchmark the project's scale or latency claims.

## 1. Build the lab, starting with Docker and Go

Download the companion's [ZIP archive](https://github.com/erain/substrate-ax-firsthand/archive/refs/heads/main.zip) in your browser and extract it. No Git installation is necessary. In a terminal, enter the extracted directory and run:

```sh
cd substrate-ax-firsthand-main
go run -buildvcs=false ./cmd/lab setup
```

The `-buildvcs=false` flag lets Go build the launcher without needing Git on your host. The launcher supplies the other tools in a private Docker container, then guides you through six stages:

```text
[1/6] Provision the private development toolbox
[2/6] Fetch the pinned Substrate and AX source
[3/6] Create Kind and connect the local image registry
[4/6] Install Substrate, snapshot storage, and gVisor workers
[5/6] Build the pinned CLIs, AX server, runner, and counter images
[6/6] Install AX and tutorial fixtures; wait for readiness
Lab ready ...
```

Here's what appeared: **Kind** runs Kubernetes nodes as Docker containers; the local registry supplies their images. **Substrate** adds the actor lifecycle and ready workers, with local snapshot storage. **AX** adds its own task API and control plane. The toolbox is just development tooling, not another execution layer for your actors.

The toolbox uses your Docker socket, so run trusted code on a private development machine. It keeps a private kubeconfig in `.local/kubeconfig` and does not switch your other contexts. It refuses to replace an existing cluster named `firsthand`. Keep this directory: it holds the lab's tools and connection state.

Check the three layers before proceeding:

```sh
./scripts/kube get nodes
./scripts/kube get pods -n ate-system
./scripts/kube get pods -n firsthand-lab
./scripts/kube get pods -n firsthand-ax
```

You should see a Ready node, healthy Substrate services, a ready worker, and the AX server and Redis running. A bucket-initialization Job marked `Completed` is normal. These wrappers use the toolbox's tools, so host kubectl still isn't required. Generated YAML is in `.local/rendered/`; inspect it whenever you're curious. If setup fails, its detailed log is `.local/setup.log`; the companion [recovery notes](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/TROUBLESHOOTING.md) explain how to resume without deleting the cluster.

## 2. Suspend a process that remembers

In a second terminal, from the companion directory, leave this running:

```sh
./scripts/router
```

It forwards the Substrate router to `127.0.0.1:18080`. Keep that terminal open.

Look at `.local/rendered/counter-template.yaml`. The important part is:

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

An ActorTemplate names the application image and its execution configuration. The WorkerPool names the worker runtime image and provisions capacity. The selector connects them. **The actor's application image is not the worker's runtime image.**

Create a counter:

```sh
./scripts/ate create actor counter-one -a firsthand --template counter
./scripts/ate get actor counter-one -a firsthand -o yaml
./scripts/kube get pods -n firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

The actor starts suspended. Note the worker Pod UID and restart count.

Our tiny counter increments two values on every POST: one held only in process memory, the other saved in `/data/count.txt`.

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

The wrapper sends an HTTP POST with `ate-target-actor: firsthand/counter-one` through that router. Its curl also lives in the toolbox. The request activated the actor. Now suspend it and inspect the checkpoint:

```sh
./scripts/ate suspend actor counter-one -a firsthand
./scripts/ate get actor counter-one -a firsthand -o yaml
```

Look for `ACTOR_STATE_SUSPENDED` and `status.externalSnapshot`. Its content scope is `FULL`, and its URI identifies the stored snapshot. In this Kind lab, local storage handles that logical `gs://` location; it does not require your own Google Cloud bucket.

**Before running the next command, predict the result.** If a fresh process simply read the file, what would happen to the memory counter?

```sh
hit
./scripts/kube get pods -n firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

Expected: `{"fileCount":3,"memoryCount":3}`, with the same Pod UID and restart count.

This isn't application-level reconstruction: our counter deliberately starts its memory value at zero and never initializes it from the file. A fresh process with the saved file would return `memoryCount:1`. The restored process returns `3`.

That's the first aha: **the Pod is capacity; the actor is the stateful execution you can suspend and restore inside it.** The incoming request wakes it without an explicit resume command.

Release the counter's active capacity before the next experiment:

```sh
./scripts/ate suspend actor counter-one -a firsthand
```

## 3. Give AX a task and a workspace

Open `.local/rendered/ax-task.yaml`. It contains just a Workspace and a Task:

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

Use the generated file; the image line above is an explanatory placeholder. Apply it through **AX**, not `kubectl`:

```sh
./scripts/ax apply -f .local/rendered/ax-task.yaml
./scripts/ax resume task task-one -a firsthand
```

This pinned version creates tasks suspended, hence the explicit resume. We leave `spec.command` empty and run the experiment ourselves after activation.

Compute a result inside the sandbox, and create a second file outside the workspace:

```sh
./scripts/ax ssh task-one -a firsthand -- sh -ec '
  printf "%s\n" 10 20 30 | awk "{total+=\$1} END {print total}" > /workspace/result.txt
  echo transient > /tmp/firsthand-marker
  cat /workspace/result.txt
'
```

You should see `60`. AX's `ssh` command executes in the running guest; we're not installing an SSH daemon or provisioning another Pod.

**Prediction time: which file will survive?**

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

This is the second aha: **“resume” doesn't always mean preserving the process.** AX's generated template uses `DATA` snapshots: the durable workspace returns, but the runtime starts fresh. Our `/tmp` marker was in the non-durable writable image layer.

Contrast the two experiments:

| Experiment | Snapshot | What survived |
|---|---|---|
| Substrate counter | FULL | RAM counter and durable file |
| AX task | DATA | Durable workspace result, not the `/tmp` marker |

Peek beneath AX to see its Substrate actor and generated template:

```sh
./scripts/ate get actor task-one -a firsthand -o yaml
./scripts/ate get actor-templates -a firsthand
./scripts/ax suspend task task-one -a firsthand
```

One version-specific caveat: AX's template in this pin doesn't select a dedicated worker pool or forward Task resource limits. Don't assume this Task used our counter's worker Pod. And `Running` describes the live runner, not proof that your command completed successfully.

## What you know now

You have created an actor, activated it by request, executed code, suspended it, and restored it. Then you declared an AX task and saw exactly what its workspace-oriented resume preserves.

We used a counter and a tiny calculation because they make state unambiguous. This is not yet an autonomous AI agent. An agent harness and real model calls are the next layer; knowing what persists first makes that layer much easier to reason about.

When finished, stop the router with Ctrl-C and review the companion [cleanup guide](https://github.com/erain/substrate-ax-firsthand/blob/main/docs/CLEANUP.md):

```sh
CONFIRM_TUTORIAL_CLEANUP=yes ./bin/lab cleanup
```

That removes the named tutorial fixtures, not other labs or the shared cluster. It doesn't purge stored snapshot bytes. Both projects are evolving quickly, so keep the pinned versions, inspect the YAML, and let the observed state—not a lifecycle verb—tell you what happened.
