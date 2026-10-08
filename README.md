# So you want to get started with Agent Substrate and AX?

Two small experiments, two observable differences:

1. **Substrate:** suspend an HTTP counter, then wake it with a request. Its RAM counter and file counter both continue; the worker Pod UID stays the same.
2. **AX:** create a Task with a Workspace, write a result, suspend, and resume. The workspace result survives; a file outside the durable workspace does not.

Start with **Docker and Go**; setup is included below, with no host Kind, kubectl, Git, Make, curl, or jq installation required. Allow **10–20 minutes for the experiments plus initial downloads/builds**. This is a local Linux/amd64 tutorial, not a production deployment or a performance benchmark. No model account, API key, or paid model call is required.

The [article draft](article/substack.md) tells the story. This README is the runnable version. Both projects are pre-stable; see [versions.env](versions.env) and the [compatibility note](docs/SETUP.md#why-is-there-an-ax-patch).

## 0. Install the complete lab

Download and extract the [repository ZIP](https://github.com/erain/substrate-ax-firsthand/archive/refs/heads/main.zip), or clone [erain/substrate-ax-firsthand](https://github.com/erain/substrate-ax-firsthand) if you already use Git, then work from its root.

With a running local Docker daemon and modern Go (automatic toolchain downloads enabled), run:

```sh
go run -buildvcs=false ./cmd/lab setup
```

The launcher reports six stages: private tool container; pinned sources; Kind/registry; Substrate/storage/workers; built CLIs and images; AX and tutorial fixtures. It waits for readiness and saves detailed output in `.local/setup.log`. Kind, kubectl, Git, Make, curl, and jq are provided inside the toolbox, not installed globally. It mounts the local Docker socket; only run this on a trusted development machine.

Verify the result:

```sh
./scripts/kube get nodes
./scripts/kube get pods -n ate-system
./scripts/kube get pods -n firsthand-lab
./scripts/kube get pods -n firsthand-ax
```

Expect a Ready Kind node and Running/ready Substrate workers and AX services. Your actors have not yet been created. The private kubeconfig is `.local/kubeconfig`; your other kubeconfig contexts are unchanged. The launcher refuses unknown existing clusters and preserves an earlier prepared lab instead of silently retargeting it. After a failed installation, diagnose the log and rerun the setup command to resume owned stages. See [SETUP.md](docs/SETUP.md) for details and the optional existing-lab route.

Builds push only to the local registry and produce digest-pinned YAML in `.local/rendered/`. Keep the companion directory in place: the persistent toolbox mounts it. Moving it is not supported.

The wrappers `scripts/ate`, `scripts/ax`, and `scripts/kube` use the context saved by `prepare`. `scripts/ax` targets the tutorial's own AX server in `firsthand-ax`, with a private tunnel cache in `.local/ax/` and no inherited `AX_SERVER` override. Switching your current kubectl context does not retarget these wrappers.

## 1. A process that remembers

Start the router forward in **a second terminal**, from this repository:

```sh
./scripts/router
```

Keep it running. It listens only on `127.0.0.1:18080`. If that port is occupied, select another during setup, for example `go run -buildvcs=false ./cmd/lab --port 28080 setup`; the router and request wrappers then use that saved port. Do not stop another person's port-forward.

In your working terminal, inspect the two manifests:

```sh
sed -n '1,100p' .local/rendered/worker-pool.yaml
sed -n '1,120p' .local/rendered/counter-template.yaml
```

The WorkerPool is a Kubernetes resource with a **worker runtime image**. The ActorTemplate is a Substrate API resource with the **counter application image**, a worker label selector, a durable `/data` directory, and `FULL` snapshots. They are different resources with different owners; do not `kubectl apply` the ActorTemplate.

Create an actor and check that it starts suspended:

```sh
./scripts/ate create actor counter-one -a firsthand --template counter
./scripts/ate get actor counter-one -a firsthand -o yaml
```

Record the worker identity, including its UID and restart count:

```sh
./scripts/kube get pods -n firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

Define a request and call it twice:

```sh
hit() {
  ./scripts/request
}
hit
hit
```

Expected:

```json
{"fileCount":1,"memoryCount":1}
{"fileCount":2,"memoryCount":2}
```

`scripts/request` runs curl in the toolbox: an HTTP POST to the forwarded router with `ate-target-actor: firsthand/counter-one`. The first request activates the suspended actor. Inspect the now-running actor, then suspend it:

```sh
./scripts/ate get actor counter-one -a firsthand -o yaml
./scripts/ate suspend actor counter-one -a firsthand
./scripts/ate get actor counter-one -a firsthand -o yaml
```

Find `status.state: ACTOR_STATE_SUSPENDED` and `status.externalSnapshot`. Its content scope is `FULL`; its URI identifies the stored snapshot. On Kind, the configured local storage backend handles it—you do not need a Google Cloud account just because the logical URI starts with `gs://`.

Now send another request, with **no explicit resume**:

```sh
hit
./scripts/kube get pods -n firsthand-lab \
  -l ate.dev/worker-pool=firsthand-workers \
  -o 'custom-columns=NAME:.metadata.name,UID:.metadata.uid,RESTARTS:.status.containerStatuses[0].restartCount'
```

Expected: `{"fileCount":3,"memoryCount":3}`, with the same worker Pod UID and restart count. The code deliberately does **not** reload the RAM counter from the file on startup. A new process with the same file would return `memoryCount:1`, not `3`; the unit test proves that distinction.

**Predict before continuing:** is this a new Pod, a new process initialized from a file, or a restored process? The observation supports the third: Substrate restores full actor state inside existing worker capacity. This is not a claim about exactly-once requests or what happens after a worker failure.

Suspend the counter so it releases active capacity:

```sh
./scripts/ate suspend actor counter-one -a firsthand
```

## 2. A task whose workspace survives

Inspect the AX manifest, then apply it through AX, not Kubernetes:

```sh
sed -n '1,120p' .local/rendered/ax-task.yaml
./scripts/ax apply -f .local/rendered/ax-task.yaml
./scripts/ax get task task-one -a firsthand
./scripts/ax resume task task-one -a firsthand
```

The manifest declares an empty `scratch` Workspace and a debug-enabled Task using our small runner image. This pinned AX version creates the task suspended, so resume is explicit. There is no `spec.command`: we run the experiment explicitly after activation, not during template preparation.

Create a tiny computed artifact and a transient marker:

```sh
./scripts/ax ssh task-one -a firsthand -- sh -ec '
  printf "%s\n" 10 20 30 | awk "{total+=\$1} END {print total}" > /workspace/result.txt
  echo transient > /tmp/firsthand-marker
  cat /workspace/result.txt
'
```

Expected: `60`. Here `ssh` is AX's guest-execution command, not an SSH daemon or a new Pod.

**Predict first:** will both files survive? Then check:

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

AX's generated ActorTemplate uses **DATA** snapshots. The durable workspace is restored into a fresh runtime; the writable image layer containing our `/tmp` marker is not preserved. This is not the counter's RAM-preserving `FULL` resume.

Inspect the underlying resources to connect the layers:

```sh
./scripts/ate get actor task-one -a firsthand -o yaml
./scripts/ate get actor-templates -a firsthand
./scripts/ax suspend task task-one -a firsthand
```

At this pin, the AX-generated template does not specify a dedicated worker selector or forward Task resource limits. Do not infer those guarantees from the manifest or assume the AX task ran in `firsthand-workers`. The counter's selector is explicit; AX can use other eligible gVisor capacity in this lab.

## What you have actually demonstrated

| Layer | Your object | What the experiment observes |
|---|---|---|
| Kubernetes | WorkerPool / worker Pod | Ready sandbox capacity outlives actor activations |
| Substrate | ActorTemplate / Actor | Request activation and full-state suspend/restore |
| AX | Workspace / Task | Higher-level declaration and workspace-preserving suspend/restore |

A `Model` is another AX primitive, but we intentionally do not exercise it here. This is an introduction to execution and state, not an autonomous LLM agent, credential tutorial, density measurement, or complete API tour.

## Tests, recovery, and cleanup

```sh
./bin/lab test      # unit tests, race detector, vet, shell syntax, safety guards
./bin/lab smoke     # end-to-end assertions; requires unused tutorial names
```

If you have followed the manual walkthrough, the smoke test will refuse to reuse your actors. It is intended for a clean tutorial scope, not to overwrite an existing run. See [validation](docs/VALIDATION.md) and [troubleshooting](docs/TROUBLESHOOTING.md).

When finished, suspend both workloads, stop the router terminal with Ctrl-C, then review [CLEANUP.md](docs/CLEANUP.md):

```sh
CONFIRM_TUTORIAL_CLEANUP=yes ./bin/lab cleanup
```

Cleanup deletes the named tutorial fixtures and their live state, not your other labs, shared Substrate, cluster, registry, or snapshot bucket. It is not a complete storage purge. Re-running the setup command prepares another walkthrough without recreating the owned cluster.

See [NOTICE](NOTICE) for upstream attribution and [the article publishing checklist](article/PUBLISHING.md) before publishing the Substack draft.
