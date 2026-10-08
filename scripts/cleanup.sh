#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
test "${CONFIRM_TUTORIAL_CLEANUP:-}" = yes || die "This deletes tutorial task/counter state. Review docs/CLEANUP.md, then run CONFIRM_TUTORIAL_CLEANUP=yes make cleanup."
load_state
for namespace in "$TUTORIAL_NAMESPACE" "$TUTORIAL_AX_NAMESPACE"; do
  owner=$(k get namespace "$namespace" -o jsonpath='{.metadata.labels.firsthand-owner}')
  test "$owner" = substrate-ax-tutorial || die "Ownership mismatch for $namespace; stopping."
done
# A connectivity error must not be mistaken for an absent task before Redis deletion.
ax get tasks -a "$TUTORIAL_ATESPACE" >/dev/null
ax get workspaces -a "$TUTORIAL_ATESPACE" >/dev/null
ate get actors -a "$TUTORIAL_ATESPACE" >/dev/null
if ax get task task-one -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then ax delete task task-one -a "$TUTORIAL_ATESPACE"; fi
if ax get workspace scratch -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then ax delete workspace scratch -a "$TUTORIAL_ATESPACE"; fi
if ate get actor counter-one -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then ate delete actor counter-one -a "$TUTORIAL_ATESPACE" --any-state; fi
if ate get actor-template counter -a "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then ate delete actor-template counter -a "$TUTORIAL_ATESPACE"; fi
ax tunnel stop "$TUTORIAL_CONTEXT"
# No namespace, atespace, tag, cluster, registry, or shared service deletion.
k delete workerpool firsthand-workers -n "$TUTORIAL_NAMESPACE" --ignore-not-found
k delete deployment ax-server firsthand-redis -n "$TUTORIAL_AX_NAMESPACE" --ignore-not-found
k delete service ax-server firsthand-redis -n "$TUTORIAL_AX_NAMESPACE" --ignore-not-found
echo "Tutorial objects removed. Shared Substrate, other labs, registry and storage are preserved. Empty namespaces remain."
