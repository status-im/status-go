#!/usr/bin/env bash
# Puts prebuilt Nim compilers and nimble on PATH of an Apple Silicon runner.
# Every download is checked against a pinned checksum.
#
#   NIM_VERSION         Nim on PATH
#   TKL_NIM_VERSION     a second Nim, TKL_NIM points at it; empty for none
#   NIMBLE_VERSION      nimble on PATH
#   NIMBLE_NIM_VERSION  the Nim nimble finds in its own directory
set -euo pipefail

if [ "$RUNNER_OS-$RUNNER_ARCH" != macOS-ARM64 ]; then
  echo "No pinned downloads for $RUNNER_OS-$RUNNER_ARCH" >&2
  exit 1
fi

tools="$RUNNER_TEMP/nim-toolchain"
nightlies=https://github.com/nim-lang/nightlies/releases/download

fetch() { # <directory> <sha256> <url> [tar options]
  local dir="$1" sum="$2" url="$3"
  shift 3
  mkdir -p "$dir"
  curl -sSfL --retry 5 --retry-all-errors -o "$RUNNER_TEMP/download" "$url"
  echo "$sum  $RUNNER_TEMP/download" | shasum -a 256 -c -
  tar -xf "$RUNNER_TEMP/download" -C "$dir" "$@"
}

install_nim() { # <version> <directory>
  case "$1" in
    2.2.4)
      fetch "$2" 86391e69225feae2c8ea54eadf560307aaf35b9b8f0bf7df40ffa84fceb2ee74 \
        "$nightlies/2025-04-22-version-2-2-f7145dd26efeeeb6eeae6fff649db244d81b212d/nim-2.2.4-macosx_arm64.tar.xz" \
        --strip-components=1 ;;
    2.2.6)
      fetch "$2" a62ff3c34a2eb3fc69866fcae05b32dc5c2b6668ecfb1d10d91ad72c0183fae5 \
        "$nightlies/2025-10-31-version-2-2-ab00c56904e3126ad826bb520d243513a139436a/nim-2.2.6-macosx_arm64.tar.xz" \
        --strip-components=1 ;;
    2.2.10)
      fetch "$2" 9a3b012d0680d11d6163dd2f145470b090c1045f5e634f42daf119bea1cb2b5e \
        "$nightlies/2026-04-24-version-2-2-bfeb3146d1638b39f69007a4ae5a23e23ae4e5ef/nim-2.2.10-macosx_arm64.tar.xz" \
        --strip-components=1 ;;
    *)
      echo "No pinned download for Nim $1" >&2
      exit 1 ;;
  esac
}

install_nim "$NIM_VERSION" "$tools/nim-$NIM_VERSION"
if [ -n "${TKL_NIM_VERSION:-}" ]; then
  [ -d "$tools/nim-$TKL_NIM_VERSION" ] || install_nim "$TKL_NIM_VERSION" "$tools/nim-$TKL_NIM_VERSION"
  echo "TKL_NIM=$tools/nim-$TKL_NIM_VERSION/bin/nim" >> "$GITHUB_ENV"
fi
install_nim "$NIMBLE_NIM_VERSION" "$HOME/.nimble/nimbinaries/nim-$NIMBLE_NIM_VERSION"

case "$NIMBLE_VERSION" in
  0.26.1)
    fetch "$tools/nimble/bin" ec27b85cd743dee9814b34811cece487b53cb19a3b696e72c06a9bc4c9a18c42 \
      "https://github.com/nim-lang/nimble/releases/download/v$NIMBLE_VERSION/nimble-macosx_aarch64.tar.gz" ;;
  *)
    echo "No pinned download for nimble $NIMBLE_VERSION" >&2
    exit 1 ;;
esac

# Nim ships an older nimble next to the compiler; the path added last comes first.
echo "$tools/nim-$NIM_VERSION/bin" >> "$GITHUB_PATH"
echo "$tools/nimble/bin" >> "$GITHUB_PATH"
