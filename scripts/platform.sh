#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
cd "$TUTORIAL_ROOT/.cache/substrate"
KIND_CLUSTER_NAME=firsthand KUBECTL_CONTEXT=kind-firsthand bash hack/install-ate-kind.sh --deploy-ate-system --credential-provider='{"enabled":false}' --rollout-timeout=240s
KIND_CLUSTER_NAME=firsthand KUBECTL_CONTEXT=kind-firsthand bash hack/install-ate-kind.sh --deploy-demo-counter --rollout-timeout=240s
