#!/usr/bin/env bash
# Source this file to set up environment variables for running tests with gotestsum.
# Usage: source ./scripts/set-test-env.sh
# NOTE: This script is for non-Nix environments only. If using Nix (nix develop),
#       the environment is already configured by the shellHook.

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "This script must be sourced, not executed."
  echo "Usage: source ./scripts/set-test-env.sh"
  exit 1
fi

if [[ -n "${IN_NIX_SHELL:-}" ]]; then
  echo "You are in a Nix environment. This script is not needed."
  echo "The Nix shellHook already configured the library paths, NIM_SDS_*, and LOGOS_STORAGE_* variables."
  return 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export LOGOS_STORAGE_SOURCE_DIR="${LOGOS_STORAGE_SOURCE_DIR:-$(realpath "$SCRIPT_DIR/../../logos-storage-nim")}"
export LOGOS_STORAGE_LIB_DIR="${LOGOS_STORAGE_LIB_DIR:-$LOGOS_STORAGE_SOURCE_DIR/build}"
export LOGOS_STORAGE_INC_DIR="${LOGOS_STORAGE_INC_DIR:-$LOGOS_STORAGE_SOURCE_DIR/library}"
export NIM_SDS_LIB_DIR="${NIM_SDS_LIB_DIR:-$(realpath "$SCRIPT_DIR/../../nim-sds/build")}"
export NIM_SDS_INC_DIR="${NIM_SDS_INC_DIR:-$(realpath "$SCRIPT_DIR/../../nim-sds/library")}"

export CGO_CFLAGS="-I$LOGOS_STORAGE_INC_DIR -I$NIM_SDS_INC_DIR"
export CGO_LDFLAGS="-L$LOGOS_STORAGE_LIB_DIR -lstorage -Wl,-rpath,$LOGOS_STORAGE_LIB_DIR -L$NIM_SDS_LIB_DIR -lsds"

if [[ "$OSTYPE" == "darwin"* ]]; then
  export DYLD_LIBRARY_PATH="$LOGOS_STORAGE_LIB_DIR:$NIM_SDS_LIB_DIR:${DYLD_LIBRARY_PATH:-}"
  echo "Using native environment (macOS)"
else
  export LD_LIBRARY_PATH="$LOGOS_STORAGE_LIB_DIR:$NIM_SDS_LIB_DIR:${LD_LIBRARY_PATH:-}"
  echo "Using native environment (Linux)"
fi

if [[ ! -f "$LOGOS_STORAGE_LIB_DIR/libstorage.so" && ! -f "$LOGOS_STORAGE_LIB_DIR/libstorage.dylib" && ! -f "$LOGOS_STORAGE_LIB_DIR/libstorage.dll" ]]; then
  echo "Warning: libstorage shared library not found in $LOGOS_STORAGE_LIB_DIR"
fi
if [[ ! -f "$NIM_SDS_LIB_DIR/libsds.so" && ! -f "$NIM_SDS_LIB_DIR/libsds.dylib" && ! -f "$NIM_SDS_LIB_DIR/libsds.dll" ]]; then
  echo "Warning: libsds shared library not found in $NIM_SDS_LIB_DIR"
fi

echo "Test environment variables set:"
echo "  LOGOS_STORAGE_SOURCE_DIR=$LOGOS_STORAGE_SOURCE_DIR"
echo "  LOGOS_STORAGE_LIB_DIR=$LOGOS_STORAGE_LIB_DIR"
echo "  LOGOS_STORAGE_INC_DIR=$LOGOS_STORAGE_INC_DIR"
echo "  NIM_SDS_LIB_DIR=$NIM_SDS_LIB_DIR"
echo "  NIM_SDS_INC_DIR=$NIM_SDS_INC_DIR"
echo "  CGO_CFLAGS=$CGO_CFLAGS"
echo "  CGO_LDFLAGS=$CGO_LDFLAGS"
echo ""
echo "You can now run tests with gotestsum, for example:"
echo '  gotestsum --packages="./services/logosstorage" -f testname -- -count 1 -tags "gowaku_no_rln gowaku_skip_migrations"'
