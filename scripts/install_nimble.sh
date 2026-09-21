#!/usr/bin/env bash
# Builds Nimble at a commit into a directory, as nim-lang/setup-nimble-action
# does. A directory that already holds a build is reused.
#
# Usage: install_nimble.sh <commit> <bin-dir>
set -euo pipefail

commit="$1"
bin_dir="$2"

exe=nimble
if [[ "${OS:-}" == "Windows_NT" ]]; then
  exe=nimble.exe
  bin_dir="$(cygpath -m "$bin_dir")"
fi

if [[ ! -x "$bin_dir/$exe" ]]; then
  src="$(mktemp -d)"
  command -v cygpath >/dev/null && src="$(cygpath -m "$src")"
  trap 'rm -rf "$src"' EXIT

  git -C "$src" init -q
  git -C "$src" remote add origin https://github.com/nim-lang/nimble
  git -C "$src" fetch -q --depth 1 origin "$commit"
  git -C "$src" checkout -q FETCH_HEAD
  git -C "$src" -c core.longpaths=true submodule update -q --init --recursive --depth 1

  mkdir -p "$bin_dir"
  nim c -d:release --hints:off -o:"$bin_dir/$exe.tmp" "$src/src/nimble.nim"
  mv "$bin_dir/$exe.tmp" "$bin_dir/$exe"
fi

"$bin_dir/$exe" --version
