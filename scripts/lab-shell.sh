#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
load_state
test -f "$TUTORIAL_ROOT/.local/kubeconfig" || die "The lab shell needs the private kubeconfig from setup."
test -n "${TUTORIAL_AX_HOME:-}" || die "Enter through ./bin/lab shell so AX has a private tunnel cache."
export PATH="$TUTORIAL_ROOT/bin:$PATH"
export KUBECONFIG="$TUTORIAL_ROOT/.local/kubeconfig"
export AX_HOME="$TUTORIAL_AX_HOME"
export TUTORIAL_CONTEXT TUTORIAL_PORT
unset AX_SERVER

# Optional arguments use the same environment without starting an interactive shell.
if test "$#" -gt 0; then exec "$@"; fi
printf 'Lab shell: context=%s, AX namespace=%s, router port=%s\n' "$TUTORIAL_CONTEXT" "$TUTORIAL_AX_NAMESPACE" "$TUTORIAL_PORT"
printf 'Use kubectl, kubectl ate, ax, and curl here. Type exit to return to your host.\n'
export PS1='firsthand:\w\$ '
exec bash --noprofile --norc
