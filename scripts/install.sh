#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
load_state
# Refuse to adopt a namespace or actor scope whose ownership is unknown.
if ate get atespace "$TUTORIAL_ATESPACE" >/dev/null 2>&1 && ! k get namespace "$TUTORIAL_NAMESPACE" >/dev/null 2>&1; then
  die "Atespace $TUTORIAL_ATESPACE already exists without our lab namespace; use a disposable cluster."
fi
for namespace in "$TUTORIAL_NAMESPACE" "$TUTORIAL_AX_NAMESPACE"; do
  if k get namespace "$namespace" >/dev/null 2>&1; then
    owner=$(k get namespace "$namespace" -o jsonpath='{.metadata.labels.firsthand-owner}')
    test "$owner" = substrate-ax-tutorial || die "Namespace $namespace exists without our ownership label."
  fi
done
k apply -f "$TUTORIAL_ROOT/.local/rendered/lab-namespace.yaml"
if ! ate get atespace "$TUTORIAL_ATESPACE" >/dev/null 2>&1; then ate create atespace "$TUTORIAL_ATESPACE"; fi
k apply -f "$TUTORIAL_ROOT/.local/rendered/ax-control-plane.yaml"
k rollout status deployment/firsthand-redis -n "$TUTORIAL_AX_NAMESPACE" --timeout=180s
k rollout status deployment/ax-server -n "$TUTORIAL_AX_NAMESPACE" --timeout=180s
ax get tasks -a "$TUTORIAL_ATESPACE"
echo "Platform ready. Enter ./bin/lab shell and create the counter's WorkerPool and ActorTemplate as shown in the article."
