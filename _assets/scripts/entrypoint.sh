#!/bin/sh

set -e

# Define $SCAN_WAKU_FLEET to run a command before running the actual app (CMD).
# This is expected to be used in functional tests to run `scan_waku_fleet.py` script before starting the app.
$SCAN_WAKU_FLEET

# This will exec the CMD from your Dockerfile, i.e. "npm start"
exec "$@"