#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
for tool in go docker kubectl git; do need "$tool"; done
test "$(uname -s)/$(uname -m)" = Linux/x86_64 || die "This edition is tested only on Linux/amd64."
test "$TUTORIAL_CONTEXT" = kind-firsthand || die "Bootstrap creates only kind-firsthand; use prepare for an existing cluster."
if kubectl config get-contexts -o name | awk '$0=="kind-firsthand" {found=1} END {exit !found}'; then
  die "kind-firsthand already exists. Bootstrap refuses to replace an existing cluster."
fi
if docker inspect firsthand-control-plane >/dev/null 2>&1; then die "firsthand-control-plane already exists; refusing replacement."; fi
if docker inspect kind-registry >/dev/null 2>&1; then
  docker port kind-registry 5000/tcp | awk '/:5001$/ {found=1} END {exit !found}' || die "Existing kind-registry is not on port 5001; refusing upstream registry replacement."
fi
bash "$TUTORIAL_ROOT/scripts/sources.sh"
cd "$TUTORIAL_ROOT/.cache/substrate"
KIND_CLUSTER_NAME=firsthand IP_FAMILY=ipv4 bash hack/create-kind-cluster.sh
if test "${1:-}" = --cluster-only; then exit 0; fi
bash "$TUTORIAL_ROOT/scripts/platform.sh"
echo "Substrate ready. Continue with make prepare and make install."
