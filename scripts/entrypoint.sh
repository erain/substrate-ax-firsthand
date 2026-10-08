#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
load_state
action=$1
shift
case "$action" in
  ate) exec "$TUTORIAL_ROOT/bin/kubectl-ate" --context "$TUTORIAL_CONTEXT" "$@" ;;
  ax)
    export AX_HOME="${TUTORIAL_AX_HOME:-$TUTORIAL_ROOT/.local/ax}"
    unset AX_SERVER
    exec "$TUTORIAL_ROOT/bin/ax" --context "$TUTORIAL_CONTEXT" --namespace "$TUTORIAL_AX_NAMESPACE" "$@"
    ;;
  kube) exec kubectl --context "$TUTORIAL_CONTEXT" "$@" ;;
  router) exec kubectl --context "$TUTORIAL_CONTEXT" port-forward -n ate-system svc/atenet-router "$TUTORIAL_PORT:80" --address=127.0.0.1 ;;
  request) exec curl -fsS --max-time 60 -X POST -H 'ate-target-actor: firsthand/counter-one' "http://127.0.0.1:$TUTORIAL_PORT/" ;;
  *) die "Unknown command wrapper: $action" ;;
esac
