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
tkl_library_dir="$(bash "$repo/scripts/tkl_native.sh")"
export NIM_TKL_INC_DIR="${NIM_TKL_INC_DIR:-$repo/build/deps/nim-token-lists/abi}"
export NIM_TKL_LIB_DIR="$tkl_library_dir"
# Go's cache does not inspect external archives. Change a cgo input whenever
# either native artifact changes, including when supplied by a Nix derivation.
tkl_build_id="$("${tkl_sha256[@]}" "$NIM_TKL_LIB_DIR/libtkl.a" "$NIM_TKL_INC_DIR/tkl.h" | "${tkl_sha256[@]}" | awk '{print $1}')"
# Native Windows Go invokes gcc outside MSYS; convert paths before constructing
# flags, and quote the whole token for Go's quoted.Split parser.
if command -v cygpath >/dev/null 2>&1; then
  NIM_TKL_INC_DIR="$(cygpath -am "$NIM_TKL_INC_DIR")"
  NIM_TKL_LIB_DIR="$(cygpath -am "$NIM_TKL_LIB_DIR")"
fi
export CGO_CFLAGS="${CGO_CFLAGS:-} \"-I$NIM_TKL_INC_DIR\" -DTKL_BUILD_ID=0x$tkl_build_id"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} \"-L$NIM_TKL_LIB_DIR\""
export GOWORK=off
if [ "$(go env GOOS)" = ios ]; then
  tkl_arch="$(go env GOARCH)"
  [ "$tkl_arch" != amd64 ] || tkl_arch=x86_64
  tkl_minimum=13.0
  if [ "${IPHONE_SDK:-iphoneos}/$tkl_arch" = iphonesimulator/arm64 ]; then
    tkl_minimum=14.0
  fi
  tkl_triple="$tkl_arch-apple-ios${IOS_TARGET:-$tkl_minimum}"
  [ "${IPHONE_SDK:-iphoneos}" != iphonesimulator ] || tkl_triple="$tkl_triple-simulator"
  tkl_sdk="$(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --show-sdk-path)"
  export CGO_CFLAGS="$CGO_CFLAGS -target $tkl_triple -isysroot \"$tkl_sdk\""
  export CGO_LDFLAGS="$CGO_LDFLAGS -target $tkl_triple -isysroot \"$tkl_sdk\""
fi
# Only the final shared library applies the private token-library export policy.
if [ "${TKL_HIDE_EXPORTS:-0}" = 1 ]; then
  case "$(go env GOOS)" in
    darwin) export CGO_LDFLAGS="$CGO_LDFLAGS \"-Wl,-unexported_symbols_list,$repo/scripts/tkl-unexported-macos.txt\"" ;;
    # cgo repeats CGO_LDFLAGS for each package at the final link. GNU ld rejects
    # repeated anonymous version scripts; archive exclusion is repeat-safe and
    # keeps libtkl private without hiding the Go library's public entry points.
    linux|android) export CGO_LDFLAGS="$CGO_LDFLAGS -Wl,--exclude-libs,libtkl.a" ;;
    windows)
      # MinGW needs exact symbol names, rather than the Apple wildcard. Derive
      # them from the pinned header, including for prebuilt Nix-style inputs.
      tkl_symbols="$(sed -n 's/^[a-z0-9_]* \(tkl_[a-z0-9_]*\)(.*/\1/p' "$NIM_TKL_INC_DIR/tkl.h")"
      [ -n "$tkl_symbols" ] || { echo "No token ABI declarations found" >&2; exit 1; }
      for tkl_symbol in $tkl_symbols; do
        export CGO_LDFLAGS="$CGO_LDFLAGS -Wl,--exclude-symbols=$tkl_symbol"
      done ;;
    *) echo "Unsupported libstatus export policy platform" >&2; exit 1 ;;
  esac
fi
cd "$repo"
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  [ "$#" -gt 0 ] || { echo "Usage: bash scripts/tkl_env.sh <command> [args...]" >&2; exit 1; }
  "$@"
fi
