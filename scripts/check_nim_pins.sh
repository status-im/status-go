#!/usr/bin/env bash
# Usage: check_nim_pins.sh [<nimble root>]
# status_go.nimble pins the Nim that builds the libraries and their revisions.
# Without an argument: the other files, which must repeat them as literals.
# With a nimble root: its resolution (nimble.paths), which a build is about to
# compile. A build may not have the other files (an image build's context).
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failed=0
miss() {
  echo "check_nim_pins.sh: $1 does not match status_go.nimble ($2)" >&2
  failed=1
}
# Fixed strings: a pin is not a pattern.
has() { grep -qF -- "$2" "$repo/$1" || miss "$1" "$2"; }

nim="$(sed -n 's|^requires "nim == \([0-9.]*\)"$|\1|p' "$repo/status_go.nimble")"
sds="$(sed -n 's|^requires "https://github.com/.*/nim-sds#\([0-9a-f]\{40\}\)"$|\1|p' "$repo/status_go.nimble")"
tkl="$(sed -n 's|^requires "https://github.com/.*/nim-token-lists#\([0-9a-f]\{40\}\)"$|\1|p' "$repo/status_go.nimble")"
delivery="$(sed -n 's|^requires "https://github.com/.*/logos-delivery#\([0-9a-f]\{40\}\)"$|\1|p' "$repo/status_go.nimble")"
[ -n "$nim" ] && [ -n "$sds" ] && [ -n "$tkl" ] && [ -n "$delivery" ] || {
  echo "check_nim_pins.sh: status_go.nimble must pin nim, nim-sds, nim-token-lists and logos-delivery" >&2
  exit 1
}

if [ "$#" -eq 0 ]; then
  has Dockerfile "ARG NIM_LIBS_VERSION=$nim"
  has nix/pkgs/libtkl/default.nix "version = \"$nim\";"
  has flake.nix "nim-sds?submodules=1&rev=$sds"
  has flake.nix "nim-token-lists?submodules=1&rev=$tkl"
  has flake.nix "logos-delivery?submodules=1&rev=$delivery"
  grep -Eq "github.com/status-im/nim-token-lists/go/tkl v[^[:space:]]*-${tkl:0:12}([[:space:]]|$)" "$repo/go.mod" ||
    miss go.mod "nim-token-lists ${tkl:0:12}"
else
  root="$1"
  # $1: package name, $2: pinned revision.
  resolved() {
    local dir rev
    dir="$(sed -n -E 's@[\\]+@/@g; s@^--path:"(.*/pkgs2/'"$1"'-[^/"]*)"[[:space:]]*$@\1@p' "$root/nimble.paths" | head -n 1)"
    [ -n "$dir" ] || { miss "$root/nimble.paths" "no $1 package"; return; }
    rev="$(sed -n 's/.*"vcsRevision": *"\([0-9a-f]*\)".*/\1/p' "$dir/nimblemeta.json" | head -n 1)"
    [ "$rev" = "$2" ] || miss "$root: $1 at ${rev:-unknown}" "$2"
  }
  resolved sds "$sds"
  resolved tokenlists "$tkl"
  resolved logos_delivery "$delivery"
fi
exit "$failed"
