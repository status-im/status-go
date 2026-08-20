#!/usr/bin/env bash
# Source this file to set up environment variables for logos-storage related tests.
# Usage: source ./scripts/mc2/set-env-for-storage.sh

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "This script must be sourced, not executed."
  echo "Usage: source ./scripts/mc2/set-env-for-storage.sh"
  exit 1
fi

GIT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && git rev-parse --show-toplevel)

# Use standard env vars when available (Nix shell sets these).
# Otherwise fall back to Makefile-style defaults.
if [[ -n "${LOGOS_STORAGE_LIB_DIR:-}" && -n "${LOGOS_STORAGE_INC_DIR:-}" ]]; then
  _LOGOS_STORAGE_LIB_DIR="${LOGOS_STORAGE_LIB_DIR}"
  _LOGOS_STORAGE_INC_DIR="${LOGOS_STORAGE_INC_DIR}"
else
  _LOGOS_STORAGE_LIB_DIR="${GIT_ROOT}/../logos-storage-nim/build"
  _LOGOS_STORAGE_INC_DIR="${GIT_ROOT}/../logos-storage-nim/library"
fi

if [[ -n "${NIM_SDS_LIB_DIR:-}" && -n "${NIM_SDS_INC_DIR:-}" ]]; then
  _NIM_SDS_LIB_DIR="${NIM_SDS_LIB_DIR}"
  _NIM_SDS_INC_DIR="${NIM_SDS_INC_DIR}"
else
  _NIM_SDS_LIB_DIR="${GIT_ROOT}/../nim-sds/build"
  _NIM_SDS_INC_DIR="${GIT_ROOT}/../nim-sds/library"
fi

# Determine shared library extension
if [[ "$OSTYPE" == "darwin"* ]]; then
  _LIB_EXT="dylib"
else
  _LIB_EXT="so"
fi

# Validate required libraries exist
if [[ ! -f "${_LOGOS_STORAGE_LIB_DIR}/libstorage.${_LIB_EXT}" ]]; then
  echo "Error: libstorage.${_LIB_EXT} not found at ${_LOGOS_STORAGE_LIB_DIR}"
  echo "Build it first (e.g. 'make build-storage') or enter the Nix shell."
  return 1
fi

if [[ ! -f "${_LOGOS_STORAGE_INC_DIR}/libstorage.h" ]]; then
  echo "Error: libstorage.h not found at ${_LOGOS_STORAGE_INC_DIR}"
  echo "Build it first (e.g. 'make build-storage') or enter the Nix shell."
  return 1
fi

if [[ ! -f "${_NIM_SDS_LIB_DIR}/libsds.${_LIB_EXT}" ]]; then
  echo "Error: libsds.${_LIB_EXT} not found at ${_NIM_SDS_LIB_DIR}"
  echo "Build it first (e.g. 'make build-libsds') or enter the Nix shell."
  return 1
fi

export LOGOS_STORAGE_LIB_DIR="${_LOGOS_STORAGE_LIB_DIR}"
export LOGOS_STORAGE_INC_DIR="${_LOGOS_STORAGE_INC_DIR}"
export NIM_SDS_LIB_DIR="${_NIM_SDS_LIB_DIR}"
export NIM_SDS_INC_DIR="${_NIM_SDS_INC_DIR}"

export CGO_CFLAGS="-I${LOGOS_STORAGE_INC_DIR} -I${NIM_SDS_INC_DIR}"
export CGO_LDFLAGS="-L${LOGOS_STORAGE_LIB_DIR} -lstorage -Wl,-rpath,${LOGOS_STORAGE_LIB_DIR} -L${NIM_SDS_LIB_DIR} -lsds"

if [[ "$OSTYPE" == "darwin"* ]]; then
  export DYLD_LIBRARY_PATH="${LOGOS_STORAGE_LIB_DIR}:${NIM_SDS_LIB_DIR}:${DYLD_LIBRARY_PATH:-}"
  echo "Using storage test environment (macOS)"
else
  export LD_LIBRARY_PATH="${LOGOS_STORAGE_LIB_DIR}:${NIM_SDS_LIB_DIR}:${LD_LIBRARY_PATH:-}"
  echo "Using storage test environment (Linux)"
fi

echo "Storage test environment variables set:"
echo "  LOGOS_STORAGE_LIB_DIR=${LOGOS_STORAGE_LIB_DIR}"
echo "  LOGOS_STORAGE_INC_DIR=${LOGOS_STORAGE_INC_DIR}"
echo "  NIM_SDS_LIB_DIR=${NIM_SDS_LIB_DIR}"
echo "  NIM_SDS_INC_DIR=${NIM_SDS_INC_DIR}"
echo "  CGO_CFLAGS=${CGO_CFLAGS}"
echo "  CGO_LDFLAGS=${CGO_LDFLAGS}"
echo ""
echo "You can now run storage tests with gotestsum, for example:"
echo '  gotestsum --packages="./pkg/services/logosstorage" -f testname -- -count 1 -tags "gowaku_no_rln use_logos_storage gowaku_skip_migrations"'
