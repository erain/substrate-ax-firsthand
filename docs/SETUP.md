# From Docker and Go to the complete lab

This edition targets a **local, disposable Linux/amd64 Docker + Kind lab**. Setup is the first part of the article. It does not claim generic Kubernetes, macOS/ARM, rootless Docker, remote Docker, cloud, or production support.

## Prerequisites

- Linux x86-64 and a local Docker daemon you can use without interactive sudo.
- Go (modern Go with automatic toolchain downloads enabled). The module uses Go 1.27.1; [Go's toolchain switching](https://go.dev/doc/toolchain) can supply it automatically.
- Internet access for pinned source revisions, Go modules, and container base images.
- Enough host resources for a Kind control plane, Substrate's supporting services, and worker capacity. Provision a generously sized development machine; this package has not established a minimum hardware requirement.
- A trusted, private development machine. This stack is experimental, has debug guest execution enabled, and is not hardened for multi-tenant or public access.

Do not use a production kubeconfig context. No Google Cloud account, bucket, model credential, or LLM billing is necessary. The local Substrate install supplies snapshot storage.

No host Kind, kubectl, Git, Make, curl, or jq installation is required. Use GitHub's Download ZIP button and extract the companion with your normal archive utility. The POSIX command wrappers run development tools inside the toolbox, not on the host.

## Main route: install the whole lab

From this repository's root:

```sh
go run -buildvcs=false ./cmd/lab setup
```

The Go launcher needs only the Docker CLI on its host. `-buildvcs=false` avoids an otherwise incidental host Git requirement. It creates a persistent toolbox container with pinned Go and kubectl, and supplies Bash, Git, Make, curl, jq, and Docker's CLI there. Kind is supplied by the pinned upstream Go tools. Setup reports stages and waits for readiness; detailed output stays in ignored `.local/setup.log`.

The toolbox mounts the repository at its original absolute path, your local Docker socket, and uses host networking to reach the local registry and Kind API. The socket gives it Docker-level control over your machine; use only trusted code on a private development host. It does not mount your user directory, original kubeconfig, model credentials, or host Docker authentication configuration.

The launcher uses `.local/kubeconfig`, not your usual kubeconfig. Cluster creation therefore does not switch your usual current context. The cached source, toolchain caches, rendered manifests, state, and a launcher binary are local artifacts. Keep this directory where it is; moving it breaks the toolbox bind mount.

`bootstrap` checks that it is not replacing an existing `firsthand-control-plane` container, then invokes the pinned Substrate Kind creation script. It enables the certificate-related Kubernetes APIs required by this version. Next, `platform` installs Substrate plus its upstream counter demo. Credential injection is disabled because this tutorial uses no model credentials. Preparation builds AX and the samples; installation deploys the independent tutorial AX control plane and fixtures.

The local Docker registry is named `kind-registry`, exposed on host loopback port 5001. It can be shared with another compatible Kind lab. The bootstrap guard refuses a differently mapped existing registry rather than allowing the upstream script to replace it. A stopped existing registry needs operator attention before bootstrap; this package will not replace it.

Installation builds source and downloads images, so timing varies. See [validation](VALIDATION.md) for measured setup results and their limitations. The article includes setup rather than sending readers here as a prerequisite.

If installation fails **after** the launcher recorded successful cluster creation, diagnose the log and rerun the same setup command. It verifies the owned cluster and resumes installation without recreating it. An unknown cluster, a partially failed creation, or a missing owned cluster is a refusal, not permission to delete and recreate. Direct `make bootstrap` is still strictly creation-only and refuses an existing cluster.

Verify installation:

```sh
./scripts/kube get nodes
./scripts/kube get pods -n ate-system
./scripts/kube get pods -n firsthand-lab
./scripts/kube get pods -n firsthand-ax
```

Expect a Ready node and healthy services and workers. The actors themselves are created later in the tutorial.

## Optional advanced route: reuse an exact compatible lab

This original host-tooling route is optional, not a reader prerequisite. It requires host Bash, Git, Make, curl, jq, kubectl, Go, and Docker. You also need the pinned Substrate install, snapshot backend, service certificate support, local registry mapping, and at least one installed gVisor WorkerPool from its counter demo. A Kubernetes cluster with only the Substrate CRD is insufficient. Use a separate companion directory from the toolbox-managed route.

```sh
export TUTORIAL_CONTEXT=kind-ate-dev  # replace with your compatible context
make prepare
make install
```

`prepare` checks the API service, registry, digest-pinned worker image, and node version. It refuses a different build label rather than guessing compatibility. It does not upgrade your cluster. Inspect `versions.env` before trying another Substrate release.

## What preparation does

1. Clones Substrate and AX into ignored `.cache/` directories and checks out exact commits from `versions.env`.
2. Applies the version-specific AX source patch below; resolves Go modules; runs AX tests; builds the CLIs, server, runner, and tutorial counter.
3. Builds images and pushes them to the **local** registry. Base image and Redis digests are pinned. The generated workload manifests use digests returned by the registry, not mutable `:lab` tags or this author's cached digests.
4. Writes context and image references to ignored `.local/state.env`; renders inspectable YAML into `.local/rendered/`.
5. `install` creates the owned namespaces `firsthand-lab` and `firsthand-ax`, the `firsthand-workers` WorkerPool, the `firsthand` atespace and `counter` ActorTemplate, and an isolated tutorial AX server with its own ephemeral Redis.

Existing unlabelled namespaces with these names are rejected. This package never grants the tutorial access to model secrets. It does not modify the existing `ax-system` server or other atespaces. Use a disposable cluster if these names already mean something to you.

The tutorial's Redis has persistence disabled. Losing it loses AX resource metadata, even if snapshot objects still exist. This is not a durable production control plane. The tutorial does not expose public ingress; router forwarding binds loopback. Cluster-internal AX/Redis traffic is teaching-lab plumbing, not a hardened multi-tenant deployment.

## Why is there an AX patch?

The pinned AX commit references older Substrate protobuf field names. Our lab runs a newer Substrate build. `compat/ax-substrate.patch` updates the wakeup-probe and snapshot-config fields, removes the obsolete `onResume.fromData` field, uses the current worker IP list, and adds an API-shape regression test.

Its relative Go module replacement resolves `../substrate` within `.cache/`; it contains no developer-specific absolute path. The patch is a local integration adjustment, **not an upstream release**, and is applied only to the tutorial's cached checkout. Original project checkouts are untouched. Do not install `ax@latest` alongside this package and expect a compatible pair.

The patch is intentionally specific, not a general compatibility layer. The scripts fail when it no longer applies and never reset a modified cached checkout. To test a different version, use a fresh companion checkout and deliberately review the new API combination.

Upstream source versions:

- [Substrate 7245baad8a6f](https://github.com/agent-substrate/substrate/tree/7245baad8a6fd58f81c32338395b74484547fc12)
- [AX ac2332829f22](https://github.com/google/ax/tree/ac2332829f22360ff97b0ba34d94dd0dd782f17e)

## Configuring local commands

For the main route, the launcher uses `kind-firsthand`, the local registry on port 5001, and router port 18080. Choose another unprivileged router port if necessary:

```sh
go run -buildvcs=false ./cmd/lab --port 28080 setup
```

The choice is saved and used by `scripts/router` and `scripts/request`. The following environment settings apply to the optional host-tooling route.

Set these **before** `make prepare`:

```sh
export TUTORIAL_CONTEXT=kind-firsthand
export TUTORIAL_PORT=18080
# The bootstrap route uses the fixed upstream loopback registry on port 5001.
export TUTORIAL_REGISTRY=localhost:5001
```

The generated state wins over later environment changes for the wrappers and install/smoke/cleanup scripts. To retarget, run preparation deliberately against the new compatible lab; do not hand-edit state to point cleanup at another cluster.

This AX pin caches tunnels by Kubernetes context, without checking the requested namespace on reuse. The AX wrapper sets its documented `AX_HOME` to ignored `.local/ax/` and removes an inherited `AX_SERVER`, so an existing tunnel to `ax-system` cannot redirect tutorial calls. Use the wrappers throughout; a bare `ax` may reach a different server.

In the main route, wrappers delegate through `bin/lab` into the persistent toolbox. This keeps AX's cached port-forwards alive between commands. `scripts/request` is a curl wrapper for the counter's routed HTTP POST, so host curl is unnecessary. Stop the router terminal with Ctrl-C when done.

The tunnel cache is additionally scoped to the toolbox's start epoch, so a toolbox restart cannot reuse stale process IDs from a previous PID namespace. If you deliberately update the toolbox Dockerfile, normal setup refuses an image mismatch. After reviewing that change, `go run -buildvcs=false ./cmd/lab --refresh-toolbox setup` replaces **only the labelled, owned toolbox**, ends its forwards, and preserves bind-mounted files and the Kind cluster. This is an explicit development upgrade path, not part of a reader's normal first setup.

`SUBSTRATE_REPOSITORY` and `AX_REPOSITORY` can point at local Git repositories to seed the cached clones. This is a developer optimization, not a reader prerequisite; revisions remain pinned and uncommitted local changes are not cloned.

For command-level learning, inspect `scripts/prepare.sh`, `scripts/install.sh`, and the generated manifests. Wrappers only choose the pinned binary, namespace, and context; they do not conceal lifecycle actions.
