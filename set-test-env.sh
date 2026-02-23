#!/usr/bin/env bash
# Source this file to set up environment variables for running tests with gotestsum
# Usage: source ./set-test-env.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export LIBS_DIR="$(realpath "$SCRIPT_DIR/libs")"
export NIM_SDS_LIB_DIR="$(realpath "$SCRIPT_DIR/../nim-sds/build")"
export NIM_SDS_INC_DIR="$(realpath "$SCRIPT_DIR/../nim-sds/library")"
export CGO_CFLAGS="-I$LIBS_DIR -I$NIM_SDS_INC_DIR"
export CGO_LDFLAGS="-L$LIBS_DIR -lstorage -Wl,-rpath,$LIBS_DIR -L$NIM_SDS_LIB_DIR -lsds"

# Detect OS and set library path accordingly
if [[ "$OSTYPE" == "darwin"* ]]; then
    export DYLD_LIBRARY_PATH="$LIBS_DIR:$NIM_SDS_LIB_DIR:$DYLD_LIBRARY_PATH"
    echo "Environment configured for macOS"
else
    export LD_LIBRARY_PATH="$LIBS_DIR:$NIM_SDS_LIB_DIR:$LD_LIBRARY_PATH"
    echo "Environment configured for Linux"
fi

echo "Test environment variables set:"
echo "  LIBS_DIR=$LIBS_DIR"
echo "  NIM_SDS_LIB_DIR=$NIM_SDS_LIB_DIR"
echo "  NIM_SDS_INC_DIR=$NIM_SDS_INC_DIR"
echo ""
echo "You can now run tests with gotestsum, for example:"
echo '  gotestsum --packages="./services/logosstorage" -f testname -- -count 1 -tags "gowaku_no_rln gowaku_skip_migrations" -run LogosStorageArchiveDownloaderSuite'
