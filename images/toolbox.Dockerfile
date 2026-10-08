# SPDX-License-Identifier: Apache-2.0
FROM golang:1.27.1-alpine3.24@sha256:cd9a32216aee5667f957a62d13a10032a63fd58e14b3f3d9cc8c2122f501e95e
RUN apk add --no-cache bash git make curl jq coreutils docker-cli docker-cli-buildx ca-certificates build-base
ARG KUBECTL_VERSION=v1.37.0
RUN curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl \
    && curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256" -o /tmp/kubectl.sha256 \
    && echo "$(cat /tmp/kubectl.sha256)  /usr/local/bin/kubectl" | sha256sum -c - \
    && chmod +x /usr/local/bin/kubectl
ARG LAB_UID=1000
ARG LAB_GID=1000
RUN if [ "$LAB_UID" != 0 ]; then \
      if ! awk -F: -v gid="$LAB_GID" '$3==gid {found=1} END {exit !found}' /etc/group; then addgroup -g "$LAB_GID" lab; fi; \
      lab_group=$(awk -F: -v gid="$LAB_GID" '$3==gid {print $1; exit}' /etc/group); \
      adduser -D -u "$LAB_UID" -G "$lab_group" lab; \
    fi
USER ${LAB_UID}:${LAB_GID}
ENV CGO_ENABLED=0 GOMAXPROCS=4 GOFLAGS=-p=4
CMD ["tail", "-f", "/dev/null"]
