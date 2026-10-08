#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
test_root=$(cd "$(dirname "$0")/.." && pwd)
export TUTORIAL_IN_TOOLBOX=1

# Exported command mocks ensure these negative tests never contact a real lab.
kubectl() {
  if test "$*" = 'config get-contexts -o name'; then
    if test "$MOCK_CASE" = existing-context; then echo kind-firsthand; fi
    return 0
  fi
  echo "Unexpected kubectl operation: $*" >&2
  return 99
}
docker() {
  case "$*" in
    'inspect firsthand-control-plane') test "$MOCK_CASE" = existing-node ;;
    'inspect kind-registry') test "$MOCK_CASE" = wrong-registry ;;
    'port kind-registry 5000/tcp') echo '127.0.0.1:5999' ;;
    *) echo "Unexpected Docker operation: $*" >&2; return 99 ;;
  esac
}
export -f kubectl docker

refuses() {
  local expected=$1 output
  shift
  if output=$("$@" 2>&1); then
    echo "Expected refusal: $expected" >&2
    exit 1
  fi
  if [[ "$output" != *"$expected"* ]]; then
    echo "Wrong refusal: $output" >&2
    exit 1
  fi
}

MOCK_CASE=existing-context TUTORIAL_CONTEXT=kind-firsthand \
  refuses 'kind-firsthand already exists' bash "$test_root/scripts/bootstrap.sh"
MOCK_CASE=existing-node TUTORIAL_CONTEXT=kind-firsthand \
  refuses 'firsthand-control-plane already exists' bash "$test_root/scripts/bootstrap.sh"
MOCK_CASE=wrong-registry TUTORIAL_CONTEXT=kind-firsthand \
  refuses 'Existing kind-registry is not on port 5001' bash "$test_root/scripts/bootstrap.sh"
MOCK_CASE=none TUTORIAL_CONTEXT=kind-unrelated \
  refuses 'Bootstrap creates only kind-firsthand' bash "$test_root/scripts/bootstrap.sh"
MOCK_CASE=none CONFIRM_TUTORIAL_CLEANUP=no \
  refuses 'This deletes tutorial task/counter state' bash "$test_root/scripts/cleanup.sh"
echo 'PASS: bootstrap replacement and unconfirmed cleanup guards fail closed.'
