#!/usr/bin/env bash
# Source this file to set up environment variables for running tests with gotestsum
# Usage: source ./scripts/set-test-env.sh
# NOTE: This script is for non-Nix environments only. If using Nix (nix develop),
#       the environment is already configured by the shellHook.

# Check if we're in a Nix shell and warn the user
if [ -n "$IN_NIX_SHELL" ]; then
    echo "⚠️  You are in a Nix environment. This script is not needed."
    echo "   The Nix shellHook has already configured CGO_CFLAGS and CGO_LDFLAGS."
    echo "   You can run tests directly without sourcing this script."
    return 0
fi

# Use local nim-sds build
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export NIM_SDS_LIB_DIR="$(realpath "$SCRIPT_DIR/../../nim-sds/build")"
export NIM_SDS_INC_DIR="$(realpath "$SCRIPT_DIR/../../nim-sds/library")"
export CGO_CFLAGS="-I$NIM_SDS_INC_DIR"
export CGO_LDFLAGS="-L$NIM_SDS_LIB_DIR -lsds"

# Detect OS and set library path accordingly
if [[ "$OSTYPE" == "darwin"* ]]; then
    export DYLD_LIBRARY_PATH="$NIM_SDS_LIB_DIR:$DYLD_LIBRARY_PATH"
    echo "✓ Using native environment (macOS)"
else
    export LD_LIBRARY_PATH="$NIM_SDS_LIB_DIR:$LD_LIBRARY_PATH"
    echo "✓ Using native environment (Linux)"
fi

echo "Test environment variables set:"
echo "  NIM_SDS_LIB_DIR=$NIM_SDS_LIB_DIR"
echo "  NIM_SDS_INC_DIR=$NIM_SDS_INC_DIR"
echo ""
echo "You can now run tests with gotestsum, for example:"
echo '  gotestsum --packages="./protocol/communities" -f testname -- -count 1 -tags "gowaku_no_rln gowaku_skip_migrations" -run "TestManagerSuite"'
