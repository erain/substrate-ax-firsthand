#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
load_state
# This test creates exactly these disposable identities and refuses to reuse them.
for actor in counter-one task-one; do
  if ate get actor "$actor" -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then die "$actor already exists; clean up tutorial objects before smoke testing."; fi
done
if ax get task task-one -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then die "task-one metadata already exists; refusing to overwrite it."; fi
if ax get workspace scratch -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then die "scratch already exists; refusing to adopt it."; fi
router_pid=
finish() {
  if test -n "$router_pid"; then kill "$router_pid" 2>/dev/null || true; wait "$router_pid" 2>/dev/null || true; fi
}
trap finish EXIT
k port-forward -n ate-system svc/atenet-router "$TUTORIAL_PORT:80" --address=127.0.0.1 > "$TUTORIAL_ROOT/.local/router.log" 2>&1 &
router_pid=$!
for attempt in {1..30}; do
  kill -0 "$router_pid" 2>/dev/null || die "Router forward failed; check .local/router.log (port may be occupied)."
  if curl -sS -o /dev/null "http://127.0.0.1:$TUTORIAL_PORT/" 2>/dev/null; then break; fi
  test "$attempt" != 30 || die "Router did not start."
  sleep 1
done
ate create actor counter-one -a "$TUTORIAL_ATESPACE" --template counter
uid_before=$(k get pods -n "$TUTORIAL_NAMESPACE" -l ate.dev/worker-pool=firsthand-workers -o jsonpath='{.items[0].metadata.uid}')
restarts_before=$(k get pods -n "$TUTORIAL_NAMESPACE" -l ate.dev/worker-pool=firsthand-workers -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}')
hit() { curl -fsS --max-time 60 -X POST -H "ate-target-actor: $TUTORIAL_ATESPACE/counter-one" "http://127.0.0.1:$TUTORIAL_PORT/"; }
first=$(hit); jq -e '.memoryCount==1 and .fileCount==1' <<< "$first" >/dev/null
second=$(hit); jq -e '.memoryCount==2 and .fileCount==2' <<< "$second" >/dev/null
ate suspend actor counter-one -a "$TUTORIAL_ATESPACE"
third=$(hit); jq -e '.memoryCount==3 and .fileCount==3' <<< "$third" >/dev/null
uid_after=$(k get pods -n "$TUTORIAL_NAMESPACE" -l ate.dev/worker-pool=firsthand-workers -o jsonpath='{.items[0].metadata.uid}')
restarts_after=$(k get pods -n "$TUTORIAL_NAMESPACE" -l ate.dev/worker-pool=firsthand-workers -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}')
test "$uid_before" = "$uid_after" || die "Worker Pod changed during the FULL snapshot test."
test "$restarts_before" = "$restarts_after" || die "Worker container restarted during the FULL snapshot test."
ate suspend actor counter-one -a "$TUTORIAL_ATESPACE"
ax apply -f "$TUTORIAL_ROOT/.local/rendered/ax-task.yaml"
ax resume task task-one -a "$TUTORIAL_ATESPACE"
ax ssh task-one -a "$TUTORIAL_ATESPACE" -- sh -ec 'printf "%s\n" 10 20 30 | awk "{total+=\$1} END {print total}" > /workspace/result.txt; echo transient > /tmp/firsthand-marker; test "$(cat /workspace/result.txt)" = 60'
ax suspend task task-one -a "$TUTORIAL_ATESPACE"
ax resume task task-one -a "$TUTORIAL_ATESPACE"
ax ssh task-one -a "$TUTORIAL_ATESPACE" -- sh -ec 'test "$(cat /workspace/result.txt)" = 60; test ! -e /tmp/firsthand-marker; echo "AX DATA restore passed"'
ax suspend task task-one -a "$TUTORIAL_ATESPACE"
ate get actor task-one -a "$TUTORIAL_ATESPACE" -o json | jq -e '.status.externalSnapshot.contentScope == "SNAPSHOT_CONTENT_SCOPE_DATA" and (.status.externalSnapshot.snapshotUri | startswith("gs://ate-snapshots/firsthand/ax/"))' >/dev/null
echo "PASS: counter 1/1 -> 2/2 -> 3/3, unchanged worker UID; AX result 60 survives, /tmp marker does not."
echo "Objects remain suspended for inspection. Use make cleanup when ready."
