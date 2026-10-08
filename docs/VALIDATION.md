# What has been validated

Validation date: **2026-10-08**. This is evidence for the pinned pair, not a promise about future upstream versions.

The first two sections record the earlier walkthrough, which installed counter fixtures automatically and used command wrappers. The native-CLI section below covers the current walkthrough.

## Existing-lab route (earlier walkthrough)

Test host: Ubuntu 24.04, Linux/amd64, local rootful Docker 29.7.2, Go 1.27.1 toolchain, and an existing Kind Substrate lab running Kubernetes 1.37. Source versions are the full commits in `versions.env`; Substrate's node build label is `v0.3.0-69-g7245baad`.

Confirmed:

- `make test`: counter unit tests with race detection, Go vet, shell syntax, and mocked fail-closed bootstrap/cleanup guard tests pass. The negative tests never contact a cluster or Docker daemon.
- `make prepare`: fresh cached checkouts at the pinned commits; portable AX patch applied; `go mod tidy`, AX's `go test ./...`, CLI/server/runner builds, local image pushes, registry digest resolution, and YAML rendering succeed.
- `make install`: tutorial namespaces, worker pool, golden counter template, independent AX server, and Redis become ready. Initial fixture install took 5.74 seconds **on the already-prepared lab with local images**.
- `make smoke`: request activation; counter values `1/1 → 2/2 → 3/3` across a FULL suspend/restore; unchanged worker Pod UID and restart count; AX result `60` survives DATA suspend/resume while `/tmp/firsthand-marker` does not. Isolated-control-plane runs took 6.46 and 6.69 seconds on this lab. That is not a measured human tutorial duration or a latency benchmark.
- No model request or credential is involved. Existing non-tutorial actors, namespaces, and platform deployments are not changed by the tutorial install.

Manual replay detected the pinned AX CLI's context-only tunnel-cache key. The final wrapper isolates its `AX_HOME` and removes inherited `AX_SERVER` overrides. A rerun confirmed the actual `firsthand-ax` tunnel, corresponding server reconciliation logs, and `gs://ate-snapshots/firsthand/ax/` DATA snapshot prefix, not just successful Task commands. The smoke test now asserts that prefix. Preliminary runs before this isolation used the existing AX server for tutorial-only `firsthand` metadata; those disposable identities were removed without changing other lab resources.

The article's counter and AX command blocks were replayed from zsh; observed output matches the article. Kubernetes custom-column expressions are quoted for zsh as well as Bash. Scoped cleanup and fixture reinstall have also been exercised successfully. This remains an author's check, not an independent reader timing study.

The automated test leaves its own actors suspended for inspection. Cleanup and reinstall can prepare a fresh manual run; do not assume smoke-test identities are unused afterwards.

## Docker-and-Go setup route (earlier walkthrough)

The launcher was run against a fresh companion directory with no source or Go cache, no private kubeconfig, and no `firsthand` cluster. It downloaded the sources from the public repositories, built the private toolbox, created a new Kind cluster, installed Substrate and snapshot storage, built AX and the samples, installed the fixtures, and reached readiness in **7 minutes 58 seconds**.

Stage measurements: toolbox 27 seconds; pinned source 7 seconds; Kind and registry 29 seconds; Substrate/storage/workers 5 minutes 53 seconds; AX and sample builds 52 seconds; fixture installation 9 seconds. Rounding and other launcher work account for the difference from total duration.

This was **not a cold Docker host**: the existing Docker daemon already had a compatible registry, the Kind node image, and some base/build layers. Source and Go caches inside the new toolbox were fresh. Do not extrapolate the eight-minute result to every reader or call it a fully cold-machine measurement. At completion the new source directory was about 267 MB and generated/cache data about 4.5 GB, before additional race-test compilation.

Confirmed on the newly created cluster:

- Ready Kind node, healthy Substrate services, ready tutorial worker, and AX server/Redis.
- End-to-end smoke assertions pass through the toolbox wrappers, including FULL counter restoration and DATA workspace restoration.
- AX uses the new `kind-firsthand` context, its `firsthand-ax` server, and the tutorial snapshot prefix.
- A setup rerun with host `PATH` containing **only Go and Docker** succeeds in 54 seconds with warm caches, verifies rather than recreates the existing cluster, and preserves actor identities/state. This exercises the no-host-Git/kubectl/Make/jq/curl setup claim.
- The original host context remains `kind-ate-dev`, and its original cluster stays Ready. No original lab resources were removed.

The initial timing run preceded adding the C compiler needed by race tests. That compiler now lives in the toolbox, not on the host. The final toolbox was explicitly rebuilt/replaced on the same owned lab (Kind and actor state preserved), and setup succeeded again with a Go-and-Docker-only host PATH. `./bin/lab test` then passed with race detection and vet inside that toolbox. Scoped cleanup through `./bin/lab cleanup` was also verified. No completely cold final-image timing is claimed.

Toolbox unit tests also cover ownership refusal, argument preservation without a host shell, private state permissions, executable installation/replacement, and tunnel-cache separation across toolbox starts.

The manual article flow was replayed on the new cluster using `scripts/request` and the two-terminal router wrapper. It produced the expected `1/1 → 2/2 → 3/3`, unchanged worker UID/restarts, and AX `60` with no transient marker. Ctrl-C in the router terminal closed its listener; a subsequent connection failed as expected rather than leaving a background forward behind. Only disposable test actor/task/workspace identities were removed afterwards.

The final launcher also passed a Go-and-Docker-only PATH setup rerun with `--port 28080` in about one minute, retaining the same Kind cluster. This exercises the configurable router port and atomic state/binary installation in the final source revision.

## Native-CLI walkthrough

The revised setup and article were tested on the same owned `kind-firsthand` lab, after scoped cleanup of the disposable tutorial fixtures. The source, image, and toolchain caches were warm; this is not another fresh-install timing.

Confirmed:

- Setup completed in 55 seconds and left the counter WorkerPool and ActorTemplate absent. The tutorial namespaces, Substrate platform, and independent AX server were ready.
- `./bin/lab shell` supplied the real CLIs with the private kubeconfig, saved port, and toolbox-epoch AX cache. Commands were copied from the article's shell blocks, rather than replaced with wrapper calls.
- The reader's `kubectl apply`, Deployment wait/rollout, and `kubectl ate create actor-template` commands created the counter capacity and starting snapshot. The first template inspection showed an empty status; another inspection showed the golden tag, as the instructions describe.
- Direct curl requests produced `1/1 → 2/2 → 3/3` across a FULL suspend/restore. The worker Pod UID stayed the same and its restart count remained zero.
- AX apply, resume, guest execution, suspend, and resume produced `60` in the workspace and removed the transient `/tmp` marker. The underlying actor showed a DATA snapshot under `gs://ate-snapshots/firsthand/ax/`.
- Port 28080 was already occupied by an unrelated forward. The documented setup retry with `--port 28081` preserved the cluster and existing counter WorkerPool. New shells picked up the port, and the raw kubectl port-forward worked. Ctrl-C closed only that forward; the existing listener on 28080 remained untouched.
- The host kubeconfig's current context remained `kind-e2e` throughout this replay.

Unit tests cover the new shell environment, literal argument preservation, missing-private-state refusals, and the installer leaving counter resource creation to the reader. The counter fixture helper also refuses an unowned namespace before writing resources. These negative branches were exercised without a cluster or Docker daemon.

Both host `make test` and toolbox `./bin/lab test` passed, including race detection, vet, shell syntax, and fail-closed guards. After scoped cleanup and AX reinstallation, `./bin/lab smoke` created the counter fixtures through the new helper and passed the FULL/DATA assertions. Cleanup removed only the disposable tutorial identities and control plane; the cluster, shared platform, unrelated forward, and stored snapshot bytes were preserved.

The final listener check caught a smoke-test cleanup bug: backgrounding the `k` shell function left its kubectl child running. The smoke test now launches kubectl directly so its exit trap owns the actual forward process. A process-lifecycle regression test failed with the old command and passed with the fix. A real smoke rerun passed, and the test port refused connections afterwards; the unrelated listener remained running.

## Not established yet

- A completely cold Docker host with no registry, node image, or cached base/build layers, and its complete installation duration.
- A 10–20 minute manual completion time measured with an independent reader.
- macOS, ARM, remote/rootless Docker, cloud clusters, production durability or hardening, worker failure recovery, density, throughput, or latency claims.

Before publishing the Substack article, run the fresh-host route and obtain an independent manual walkthrough. Keep the article's setup-time qualification unless the distribution and timing are deliberately redesigned and validated.
