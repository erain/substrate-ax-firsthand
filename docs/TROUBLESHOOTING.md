# Troubleshooting

## Setup failed or appears to be taking a long time

The launcher prints a heartbeat every 30 seconds during a build or download, and stores detailed output in `.local/setup.log`. Open that file in your editor; host curl, jq, kubectl, and Make are not needed. Check Docker disk space, memory and network access before retrying. A first source build is different from a warmed-up actor activation.

Once the launcher has recorded successful cluster creation, rerun `go run -buildvcs=false ./cmd/lab setup` to resume installation without recreating the cluster. If it finds a cluster it doesn't recognize, or cluster creation failed partway through, it stops. Check the error before deciding what to do; don't delete another lab or bypass the guard.

## Go asks for a newer toolchain

The pinned modules use Go 1.27.1. Modern Go downloads that toolchain with its normal automatic setting. If you disabled `GOTOOLCHAIN` downloads, re-enable them for this command or install the required version. The Go launcher is standard-library-only; it does not need host Git when invoked with `-buildvcs=false`.

## Docker socket or toolbox unavailable

Check `docker info`. This edition needs a local, rootful Linux/amd64 Docker daemon that you can use without interactive sudo. Remote Docker, rootless Docker, macOS and ARM are not validated. The toolbox needs the host socket and repository bind mount, so it has Docker-level authority over your development machine. The directory must not be moved while the toolbox exists.

## AX reaches the wrong namespace

Use `./scripts/ax`, not a bare `ax`. This pin caches tunnels by context rather than by context and namespace. The wrapper isolates `AX_HOME` in `.local/ax/` and removes inherited `AX_SERVER` overrides. Check `./scripts/ax tunnel list`: the tutorial tunnel should say `firsthand-ax`. Do not stop or repurpose another lab's tunnel.

## curl cannot connect to port 18080

The router port-forward is not running, the wrong port is used, or the port is occupied. In a second terminal, from this repo, run `./scripts/router` and leave it running. Check its output before retrying `./scripts/request`. A port-forward is a local connection, not an actor or a durable service installation.

If you set a different `TUTORIAL_PORT` before preparation, use that port. Do not stop another person's port-forward to take its port.

## Counter does not return 1/1 on the first request

This probably is not a fresh `counter-one`. Requests increment state, including retries. Inspect the actor before continuing. The smoke test refuses to adopt existing names; the manual create command also fails rather than resetting state. Use the scoped cleanup procedure only if this is your disposable run.

## ActorTemplate has no golden tag

`install` waits for preparation before handing you the lab. Inspect:

```sh
./scripts/ate get actor-template counter -a firsthand -o yaml
./scripts/kube get pods -n firsthand-lab
./scripts/kube get workerpools -A
```

Golden preparation needs compatible active worker capacity and snapshot storage. Check image pull errors, node version labels, template error status, and Substrate logs. Do not infer a working install merely from CRD existence.

## AX Task is Suspended after apply

Expected for this pinned version. Run `./scripts/ax resume task task-one -a firsthand`, then inspect `./scripts/ax get task task-one -a firsthand`. The runner has no automatic application command in this demo; you execute the computation through `ax ssh` once it is ready.

## AX task is Running after my command returned

The debug runner is still running. This pin does not turn the one-off guest command's exit into Task completion. A `Running` phase is not proof that your calculation succeeded. Use the command's exit status and `result.txt`; suspend explicitly afterwards.

## /tmp marker survives

Check that you suspended and resumed the right Task through `scripts/ax`, not the counter or another AX installation. Inspect its Substrate ActorTemplate's `snapshotConfig`: the AX sample should use DATA, not FULL. Suspend does not mean keeping the original writable image layer for DATA snapshots.

## AX apply reports no available workers

Suspend the counter and other disposable workloads you own, or provision sufficient compatible worker capacity. Do not stop someone else's actors. At this pin the AX-generated template does not select our dedicated counter pool, so check all compatible gVisor pools.

## Prepare reports incompatible version or patch failure

This repository pins a tested combination. It does not upgrade Substrate or silently adapt to latest AX. Check `versions.env` and [SETUP.md](SETUP.md). Preserve changes in existing checkouts; never repair this with `git reset --hard`.

## Redis restarted and tasks disappeared

The teaching AX control plane has ephemeral Redis. Its resource metadata is not persisted. That is separate from the substrate snapshot backend. Recreate the disposable tutorial only after inspecting any surviving underlying actors; do not assume missing AX metadata proves the actor was deleted.
