#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
load_state
# Automated tests prepare the same resources that readers create explicitly.
namespace_owner=$(k get namespace "$TUTORIAL_NAMESPACE" -o jsonpath='{.metadata.labels.firsthand-owner}')
test "$namespace_owner" = substrate-ax-tutorial || die "Ownership mismatch for $TUTORIAL_NAMESPACE; stopping."
k apply -f "$TUTORIAL_ROOT/.local/rendered/worker-pool.yaml"
k wait --for=create deployment/firsthand-workers -n "$TUTORIAL_NAMESPACE" --timeout=60s
k rollout status deployment/firsthand-workers -n "$TUTORIAL_NAMESPACE" --timeout=180s
if ! ate get actor-template counter -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then
  ate create actor-template -f "$TUTORIAL_ROOT/.local/rendered/counter-template.yaml"
fi
for attempt in {1..90}; do
  status=$(ate get actor-template counter -a "$TUTORIAL_ATESPACE" -o json)
  golden=$(jq -r '.status.goldenSnapshotStatus.goldenTag.name // empty' <<< "$status")
  if test -n "$golden"; then break; fi
  test "$attempt" != 90 || die "Golden snapshot was not ready in time. Inspect the template status."
  sleep 2
done
