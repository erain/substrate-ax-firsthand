#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
for tool in go docker kubectl git jq curl; do need "$tool"; done
test "$(uname -s)/$(uname -m)" = Linux/x86_64 || die "This edition is tested only on Linux/amd64."
docker info >/dev/null
k get svc api -n ate-system >/dev/null || die "Install Substrate first; see docs/SETUP.md."
curl -fsS "http://$TUTORIAL_REGISTRY/v2/" >/dev/null || die "Local registry is not reachable."
pool=$(k get workerpools -A -o json | jq -c '[.items[]|select(.spec.sandboxClass=="gvisor")][0]')
test "$pool" != null || die "Install the gVisor counter demo first to supply a compatible worker image."
WORKER_IMAGE=$(jq -r '.spec.workerImage' <<< "$pool")
NODE_VERSION=$(jq -r '.spec.template.nodeSelector["ate.dev/substrate-version"]' <<< "$pool")
test "$NODE_VERSION" = v0.3.0-69-g7245baad || die "This tutorial pins a different Substrate build; do not mix versions."
[[ "$WORKER_IMAGE" == *@sha256:* ]] || die "Worker image must be digest-pinned."
bash "$TUTORIAL_ROOT/scripts/sources.sh"
mkdir -p "$TUTORIAL_ROOT/bin" "$TUTORIAL_ROOT/.local/rendered"
cd "$TUTORIAL_ROOT/.cache/substrate"
CGO_ENABLED=0 go build -trimpath -o "$TUTORIAL_ROOT/bin/kubectl-ate" ./cmd/kubectl-ate
cd "$TUTORIAL_ROOT/.cache/ax"
go mod tidy
go test ./...
CGO_ENABLED=0 go build -trimpath -o "$TUTORIAL_ROOT/bin/ax" ./cmd/ax
CGO_ENABLED=0 go build -trimpath -o "$TUTORIAL_ROOT/bin/ax-server" ./cmd/ax-server
CGO_ENABLED=0 go build -trimpath -o "$TUTORIAL_ROOT/bin/ax-task-runner" ./cmd/ax-task-runner
cd "$TUTORIAL_ROOT"
go test ./...
CGO_ENABLED=0 go build -trimpath -o bin/counter ./cmd/counter
build_image() {
  local name=$1 dockerfile=$2
  docker build --build-arg "BASE_STATIC=$BASE_STATIC" --build-arg "BASE_SHELL=$BASE_SHELL" -t "$TUTORIAL_REGISTRY/firsthand-$name:lab" -f "$dockerfile" .
  docker push "$TUTORIAL_REGISTRY/firsthand-$name:lab"
  # Registry response, not Docker's potentially stale multi-platform RepoDigests.
  local digest
  digest=$(curl -fsSI -H 'Accept: application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json,application/vnd.docker.distribution.manifest.v2+json' "http://$TUTORIAL_REGISTRY/v2/firsthand-$name/manifests/lab" | awk 'tolower($1)=="docker-content-digest:" {gsub("\r","",$2);print $2}')
  [[ "$digest" == sha256:* ]] || die "No registry digest for $name."
  printf -v "$3" '%s' "$TUTORIAL_REGISTRY/firsthand-$name@$digest"
}
build_image counter images/counter.Dockerfile COUNTER_IMAGE
build_image runner images/runner.Dockerfile RUNNER_IMAGE
build_image ax-server images/ax-server.Dockerfile AX_IMAGE
# Mirror the pinned amd64 Redis manifest rather than relying on a mutable tag.
docker pull redis@sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499
docker tag redis@sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499 "$TUTORIAL_REGISTRY/firsthand-redis:lab"
docker push "$TUTORIAL_REGISTRY/firsthand-redis:lab"
redis_digest=$(curl -fsSI -H 'Accept: application/vnd.oci.image.manifest.v1+json,application/vnd.docker.distribution.manifest.v2+json' "http://$TUTORIAL_REGISTRY/v2/firsthand-redis/manifests/lab" | awk 'tolower($1)=="docker-content-digest:" {gsub("\r","",$2);print $2}')
[[ "$redis_digest" == sha256:* ]] || die "No Redis manifest digest."
REDIS_IMAGE="$TUTORIAL_REGISTRY/firsthand-redis@$redis_digest"
for variable in TUTORIAL_CONTEXT TUTORIAL_REGISTRY TUTORIAL_PORT WORKER_IMAGE NODE_VERSION COUNTER_IMAGE RUNNER_IMAGE AX_IMAGE REDIS_IMAGE; do
  printf '%s=%q\n' "$variable" "${!variable}"
done > .local/state.env
render templates/lab-namespace.yaml.tmpl .local/rendered/lab-namespace.yaml
render templates/worker-pool.yaml.tmpl .local/rendered/worker-pool.yaml
render templates/counter-template.yaml.tmpl .local/rendered/counter-template.yaml
render templates/ax-control-plane.yaml.tmpl .local/rendered/ax-control-plane.yaml
render templates/ax-task.yaml.tmpl .local/rendered/ax-task.yaml
echo "Prepared digest-pinned images and rendered YAML in .local/rendered. Run make install."
