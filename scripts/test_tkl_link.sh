#!/usr/bin/env bash
# Link the public Go wrapper as a shared library without requiring SDS or Qt.
set -euo pipefail
cd "$(dirname "$0")/.."
case "$(go env GOOS)" in
  darwin) extension=dylib ;;
  linux|android) extension=so ;;
  windows) extension=dll ;;
  *) echo "Shared-library probe does not support this target" >&2; exit 1 ;;
esac
export TKL_HIDE_EXPORTS=1
source scripts/tkl_env.sh
mkdir -p build
go build -buildmode=c-shared -o "build/tkl-probe.$extension" ./scripts/testdata/tkl-probe
TKL_TARGET_OS="$(go env GOOS)" bash scripts/check_tkl_exports.sh "build/tkl-probe.$extension"
# The probe's public entry must survive; hiding all exports is not a valid fix.
case "$extension" in
  dylib) symbols="$(nm -gU build/tkl-probe.dylib)" ;;
  so) symbols="$("${NM:-nm}" -D --defined-only build/tkl-probe.so)" ;;
  dll) symbols="$("${OBJDUMP:-objdump}" -p build/tkl-probe.dll)" ;;
esac
printf '%s\n' "$symbols" | awk '$NF ~ /^_?TokenLibraryProbe$/ { found=1 } END { exit !found }'
