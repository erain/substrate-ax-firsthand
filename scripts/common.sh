#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
TUTORIAL_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
if test -f "$TUTORIAL_ROOT/.local/launcher.json" && test "${TUTORIAL_IN_TOOLBOX:-}" != 1; then
  case "$0" in /*) task_script=$0 ;; *) task_script="$TUTORIAL_ROOT/$0" ;; esac
  exec "$TUTORIAL_ROOT/bin/lab" --root "$TUTORIAL_ROOT" exec bash "$task_script" "$@"
fi
source "$TUTORIAL_ROOT/versions.env"
TUTORIAL_CONTEXT=${TUTORIAL_CONTEXT:-kind-firsthand}
TUTORIAL_REGISTRY=${TUTORIAL_REGISTRY:-localhost:5001}
TUTORIAL_PORT=${TUTORIAL_PORT:-18080}
TUTORIAL_ATESPACE=firsthand
TUTORIAL_NAMESPACE=firsthand-lab
TUTORIAL_AX_NAMESPACE=firsthand-ax

die() { echo "error: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null || die "Install $1 first."; }
load_state() {
  test -f "$TUTORIAL_ROOT/.local/state.env" || die "Run go run -buildvcs=false ./cmd/lab setup first (or make prepare for the advanced host-tooling route)."
  source "$TUTORIAL_ROOT/.local/state.env"
}
ate() { "$TUTORIAL_ROOT/bin/kubectl-ate" --context "$TUTORIAL_CONTEXT" "$@"; }
ax() { "$TUTORIAL_ROOT/scripts/ax" "$@"; }
k() { kubectl --context "$TUTORIAL_CONTEXT" "$@"; }
render() {
  local input=$1 output=$2
  sed -e "s|@WORKER_IMAGE@|$WORKER_IMAGE|g" \
      -e "s|@NODE_VERSION@|$NODE_VERSION|g" \
      -e "s|@COUNTER_IMAGE@|$COUNTER_IMAGE|g" \
      -e "s|@AX_IMAGE@|$AX_IMAGE|g" \
      -e "s|@RUNNER_IMAGE@|$RUNNER_IMAGE|g" \
      -e "s|@REDIS_IMAGE@|$REDIS_IMAGE|g" "$input" > "$output"
}
