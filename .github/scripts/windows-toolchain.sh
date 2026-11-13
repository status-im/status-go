#!/usr/bin/env bash
# Puts the compiler, nimble and protoc on PATH of a Windows runner. Runs in the
# MSYS2 shell. Every download is checked against a pinned checksum.
#
#   NIMBLE_VERSION, PROTOC_VERSION  the versions the Dockerfile pins
set -euo pipefail

# The compiler the Jenkins hosts built with (status-im/infra-scoop-bucket).
winlibs_release=15.2.0posix-13.0.0-ucrt-r5
winlibs_archive=winlibs-x86_64-posix-seh-gcc-15.2.0-mingw-w64ucrt-13.0.0-r5.7z
winlibs_sha256=6b567e3c158a00c81ceb85298487c905ced3288fb5fb17636e2471cf8134927a
# Checksums of the Windows archives of the pinned versions. A new version in the
# Dockerfile fails the download check until these follow.
nimble_sha256=e158b8e501f6b1055cc5d4fef742e1eb928b685d67842a7b4e384183079b42f3
protoc_sha256=510fb2369a4720adb457783768c5e1481a0e1137d9e7694478dfe9b8fe445dc1

tools="$(cygpath -u "$RUNNER_TEMP")/tools"
mkdir -p "$tools"

fetch() { # <url> <sha256> <file>
  curl -sSfL --retry 3 -o "$3" "$1"
  echo "$2  $3" | sha256sum -c -
}

fetch "https://github.com/brechtsanders/winlibs_mingw/releases/download/$winlibs_release/$winlibs_archive" \
  "$winlibs_sha256" "$tools/winlibs.7z"
7z x -bso0 -bsp0 "-o$(cygpath -w "$tools")" "$(cygpath -w "$tools/winlibs.7z")"
rm "$tools/winlibs.7z"
fetch "https://github.com/nim-lang/nimble/releases/download/v$NIMBLE_VERSION/nimble-windows_x64.zip" \
  "$nimble_sha256" "$tools/nimble.zip"
unzip -q "$tools/nimble.zip" -d "$tools/nimble"
fetch "https://github.com/protocolbuffers/protobuf/releases/download/v$PROTOC_VERSION/protoc-$PROTOC_VERSION-win64.zip" \
  "$protoc_sha256" "$tools/protoc.zip"
unzip -q "$tools/protoc.zip" -d "$tools/protoc"

# Ahead of the compilers the runner image ships.
cygpath -w "$tools/mingw64/bin" >> "$GITHUB_PATH"
cygpath -w "$tools/protoc/bin" >> "$GITHUB_PATH"
# Added last so that it precedes the nimble bundled with Nim.
cygpath -w "$tools/nimble" >> "$GITHUB_PATH"
# A fixed place for the code generation cache, on the runner's work disk.
echo "GO_GENERATE_FAST_DIR=$(cygpath -m "$RUNNER_TEMP")/status-go-cache/go-generate-fast" >> "$GITHUB_ENV"
