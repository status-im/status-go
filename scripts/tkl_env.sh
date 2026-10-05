#!/usr/bin/env bash
# Source this file (also enables strict mode), or execute it with a command.
# Public Go module resolution and native preparation share the repository pin.
set -euo pipefail
if command -v sha256sum >/dev/null 2>&1; then
  tkl_sha256=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  tkl_sha256=(shasum -a 256)
else
  echo "libtkl: Install sha256sum or shasum for SHA-256 checksums" >&2
  exit 1
fi
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bash "$repo/scripts/tkl_native.sh"
export NIM_TKL_INC_DIR="${NIM_TKL_INC_DIR:-$repo/build/deps/nim-token-lists/abi}"
export NIM_TKL_LIB_DIR="${NIM_TKL_LIB_DIR:-$repo/build/deps/nim-token-lists/build/link}"
# Go's cache does not inspect external archives. Change a cgo input whenever
# either native artifact changes, including when supplied by a Nix derivation.
tkl_build_id="$("${tkl_sha256[@]}" "$NIM_TKL_LIB_DIR/libtkl.a" "$NIM_TKL_INC_DIR/tkl.h" | "${tkl_sha256[@]}" | awk '{print $1}')"
export CGO_CFLAGS="${CGO_CFLAGS:-} -I$NIM_TKL_INC_DIR -DTKL_BUILD_ID=0x$tkl_build_id"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -L$NIM_TKL_LIB_DIR"
export GOWORK=off
cd "$repo"
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  [ "$#" -gt 0 ] || { echo "Usage: bash scripts/tkl_env.sh <command> [args...]" >&2; exit 1; }
  "$@"
fi
