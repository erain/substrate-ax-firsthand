#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
source "$(dirname "$0")/common.sh"
need git
mkdir -p "$TUTORIAL_ROOT/.cache"
checkout() {
  local name=$1 url=$2 revision=$3 path="$TUTORIAL_ROOT/.cache/$1"
  if test ! -d "$path"; then
    git clone "$url" "$path"
    git -C "$path" checkout --detach "$revision"
  fi
  test "$(git -C "$path" rev-parse HEAD)" = "$revision" || die "Unexpected $name revision in $path; preserve your changes and use a fresh tutorial checkout."
}
checkout substrate "${SUBSTRATE_REPOSITORY:-https://github.com/agent-substrate/substrate.git}" "$SUBSTRATE_COMMIT"
checkout ax "${AX_REPOSITORY:-https://github.com/google/ax.git}" "$AX_COMMIT"
patch="$TUTORIAL_ROOT/compat/ax-substrate.patch"
if git -C "$TUTORIAL_ROOT/.cache/ax" apply --check "$patch" 2>/dev/null; then
  git -C "$TUTORIAL_ROOT/.cache/ax" apply "$patch"
elif ! git -C "$TUTORIAL_ROOT/.cache/ax" apply --reverse --check "$patch" 2>/dev/null; then
  die "AX patch cannot be applied safely. Do not reset an existing checkout."
fi
