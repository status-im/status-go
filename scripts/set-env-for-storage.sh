#!/usr/bin/env bash
# Source this file to set up environment variables for logos-storage related tests.
# Usage: source ./scripts/set-env-for-storage.sh

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "This script must be sourced, not executed."
  echo "Usage: source ./scripts/set-env-for-storage.sh"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -n "${LIBSTORAGE_PATH:-}" ]]; then
  DEFAULT_STORAGE_LIB_DIR="$(realpath "$LIBSTORAGE_PATH/lib")"
  DEFAULT_STORAGE_INC_DIR="$(realpath "$LIBSTORAGE_PATH/include")"
else
  DEFAULT_STORAGE_LIB_DIR="$(realpath "$SCRIPT_DIR/../libs")"
  DEFAULT_STORAGE_INC_DIR="$DEFAULT_STORAGE_LIB_DIR"
fi

if [[ -n "${LIBSDS_PATH:-}" ]]; then
  DEFAULT_NIM_SDS_LIB_DIR="$(realpath "$LIBSDS_PATH/lib")"
  DEFAULT_NIM_SDS_INC_DIR="$(realpath "$LIBSDS_PATH/include")"
else
  DEFAULT_NIM_SDS_LIB_DIR="$(realpath "$SCRIPT_DIR/../../nim-sds/build")"
  DEFAULT_NIM_SDS_INC_DIR="$(realpath "$SCRIPT_DIR/../../nim-sds/library")"
fi

export LIBS_DIR="${LIBS_DIR:-$DEFAULT_STORAGE_LIB_DIR}"
export LOGOS_STORAGE_LIB_DIR="${LOGOS_STORAGE_LIB_DIR:-$DEFAULT_STORAGE_LIB_DIR}"
export LOGOS_STORAGE_INC_DIR="${LOGOS_STORAGE_INC_DIR:-$DEFAULT_STORAGE_INC_DIR}"
export NIM_SDS_LIB_DIR="${NIM_SDS_LIB_DIR:-$DEFAULT_NIM_SDS_LIB_DIR}"
export NIM_SDS_INC_DIR="${NIM_SDS_INC_DIR:-$DEFAULT_NIM_SDS_INC_DIR}"

export CGO_CFLAGS="-I$LOGOS_STORAGE_INC_DIR -I$NIM_SDS_INC_DIR"
export CGO_LDFLAGS="-L$LOGOS_STORAGE_LIB_DIR -lstorage -Wl,-rpath,$LOGOS_STORAGE_LIB_DIR -L$NIM_SDS_LIB_DIR -lsds"

if [[ "$OSTYPE" == "darwin"* ]]; then
  export DYLD_LIBRARY_PATH="$LOGOS_STORAGE_LIB_DIR:$NIM_SDS_LIB_DIR:${DYLD_LIBRARY_PATH:-}"
  echo "Using storage test environment (macOS)"
else
  export LD_LIBRARY_PATH="$LOGOS_STORAGE_LIB_DIR:$NIM_SDS_LIB_DIR:${LD_LIBRARY_PATH:-}"
  echo "Using storage test environment (Linux)"
fi

if [[ ! -f "$LOGOS_STORAGE_LIB_DIR/libstorage.so" && ! -f "$LOGOS_STORAGE_LIB_DIR/libstorage.dylib" && ! -f "$LOGOS_STORAGE_LIB_DIR/libstorage.dll" ]]; then
  echo "Warning: libstorage shared library not found in $LOGOS_STORAGE_LIB_DIR"
fi
if [[ ! -f "$LOGOS_STORAGE_INC_DIR/libstorage.h" ]]; then
  echo "Warning: libstorage.h not found in $LOGOS_STORAGE_INC_DIR"
fi
if [[ ! -f "$NIM_SDS_LIB_DIR/libsds.so" && ! -f "$NIM_SDS_LIB_DIR/libsds.dylib" && ! -f "$NIM_SDS_LIB_DIR/libsds.dll" ]]; then
  echo "Warning: libsds shared library not found in $NIM_SDS_LIB_DIR"
fi

echo "Storage test environment variables set:"
echo "  LOGOS_STORAGE_LIB_DIR=$LOGOS_STORAGE_LIB_DIR"
echo "  LOGOS_STORAGE_INC_DIR=$LOGOS_STORAGE_INC_DIR"
echo "  NIM_SDS_LIB_DIR=$NIM_SDS_LIB_DIR"
echo "  NIM_SDS_INC_DIR=$NIM_SDS_INC_DIR"
echo "  CGO_CFLAGS=$CGO_CFLAGS"
echo "  CGO_LDFLAGS=$CGO_LDFLAGS"
echo ""
echo "You can now run storage tests with gotestsum, for example:"
echo '  gotestsum --packages="./services/logosstorage" -f testname -- -count 1 -tags "gowaku_no_rln use_logos_storage gowaku_skip_migrations"'
