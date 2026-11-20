#!/usr/bin/env bash
# Puts a prebuilt, checksum-pinned Nim compiler and nimble on PATH of an Apple Silicon runner.
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
    2.2.12)
      fetch "$2" 130a43eff76c1c3cc78b89329a7efd892c1f65839dbc6d8e53651f3f77516fe5 \
        "$nightlies/2026-09-08-version-2-2-8e8fbf60693418dc95bb0d762fd660231d08a583/nim-2.2.12-macosx_arm64.tar.xz" \
        --strip-components=1 ;;
    *)
      echo "No pinned download for Nim $1" >&2
      exit 1 ;;
  esac
}

install_nim "$NIM_VERSION" "$tools/nim-$NIM_VERSION"

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
