#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create workspace (if necessary) and dump all dependencies to it
ROOT=$PWD
WS1="$ROOT/build/_workspace/deps"
WS2="$ROOT/build/_workspace/project"

# expose all vendored packages
if [ ! -d "$WS1/src" ]; then
    mkdir -p "$WS1"
    cd "$WS1"
    ln -s "$ROOT/src/vendor" src
    cd "$ROOT"
fi

# expose project itself
PROJECTDIR="$WS2/src/github.com/status-im"
if [ ! -L "$PROJECTDIR/status-go" ]; then
    mkdir -p "$PROJECTDIR"
    cd "$PROJECTDIR"
    ln -s "$ROOT" status-go
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$WS1:$WS2"
GOBIN="$PWD/build/bin"
export GOPATH GOBIN

# Run the command inside the workspace.
cd "$PROJECTDIR/status-go"

# Linker options
export CGO_CFLAGS="-I/$JAVA_HOME/include -I/$JAVA_HOME/include/darwin"

# Launch the arguments with the configured environment.
exec "$@"

