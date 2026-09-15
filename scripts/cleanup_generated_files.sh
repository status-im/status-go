#!/usr/bin/env bash
set -euo pipefail

# Removes the UNTRACKED generated files (the test-only mocks; see .gitignore).
#
# The generated sources the library build needs (*.pb.go, bindata.go,
# migrations.go, the endpoint and messenger handler tables) are committed so
# that a consumer resolving status-go as a nimble dependency gets a complete
# tree. Deleting them here would make `make generate` destructive for anyone
# without the full generator toolchain, and an orphan among them now shows up
# as a deletion candidate in `git status` instead.

DRY_RUN="${CLEANUP_GENERATED_FILES_DRY_RUN:-false}"
CMD="rm -rf"
if [[ "$DRY_RUN" == "true" ]]; then
    CMD="echo [DRY-RUN]"
fi

echo "Cleaning up generated files... (dry run: $DRY_RUN) "

# Ignoring vendor directory is required for nix builds.

# The chainutils mock is committed (a non-test file imports it), so it is not
# an untracked generated file and must survive this sweep.
find . -path './vendor' -prune -o -path './pkg/services/connector/chainutils/mock' -prune -o -type d -name 'mock' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name 'mock.go' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name '*_mock_test.go' -exec $CMD {} +
