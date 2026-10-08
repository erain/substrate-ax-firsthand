# Cleanup scope

First suspend the counter and AX task if they are still running, then stop your `scripts/router` terminal with Ctrl-C. Stopping the port-forward does not stop either workload.

Review `.local/state.env` to confirm the intended context. Cleanup requires both namespace ownership labels to match, and requires an explicit confirmation:

```sh
CONFIRM_TUTORIAL_CLEANUP=yes ./bin/lab cleanup
```

It deletes these exact tutorial identities:

- AX `Task/task-one` and `Workspace/scratch` in atespace `firsthand`; AX's task deletion also cleans its managed underlying actor and dedicated template.
- Substrate `Actor/counter-one` and `ActorTemplate/counter` in atespace `firsthand`.
- Kubernetes `WorkerPool/firsthand-workers` in `firsthand-lab`.
- The tutorial `ax-server` and `firsthand-redis` Deployments and Services in `firsthand-ax`.

It also stops this companion's private AX tunnel for the saved context, not another lab's tunnel. The dedicated AX server is reserved for this tutorial: do not run other work on it. Deleting its ephemeral Redis loses **all** metadata stored in that server, including any extra resources you chose to create there. If you repurposed it, do not run this cleanup script.

**The counter value and AX result are no longer accessible through these resources afterwards.** There is no undo command; start a new experiment or recover deliberately from a retained snapshot if you need the old data.

It preserves the namespace objects, service account, atespace, shared Substrate services, the upstream demo, other labs, the Kind cluster, local registry/images, cached source/binaries, and snapshot storage. In particular, this is **not** a purge of snapshot bytes or a complete uninstall. Do not use sensitive data in this tutorial.

The optional host-tooling route uses `CONFIRM_TUTORIAL_CLEANUP=yes make cleanup` instead. The main route uses the command above and needs no host Make.

Run the setup command again to recreate the fixtures for another walkthrough without recreating the owned cluster. The toolbox and private kubeconfig remain, as do downloaded caches. Full cluster, registry, and toolbox deletion are separate operator actions, deliberately not scripted here. Do not remove or move this directory while its toolbox is running.
