#!/usr/bin/env bash
set -euo pipefail

# Removes the UNTRACKED generated files (the test-only mocks; see .gitignore).
# The generated sources the library build needs are committed (see
# docs/building.md), so this sweep must leave them in place: an orphan among
# them shows up as a deletion candidate in `git status` instead.

DRY_RUN="${CLEANUP_GENERATED_FILES_DRY_RUN:-false}"
CMD="rm -rf"
if [[ "$DRY_RUN" == "true" ]]; then
    CMD="echo [DRY-RUN]"
fi

echo "Cleaning up generated files... (dry run: $DRY_RUN) "

# Ignoring vendor directory is required for nix builds.

find . -path './vendor' -prune -o -type d -name 'mock' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name 'mock.go' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name '*_mock_test.go' -exec $CMD {} +

$CMD ./pkg/version/VERSION
$CMD ./pkg/version/GIT_COMMIT
$CMD ./pkg/sentry/SENTRY_CONTEXT_NAME
$CMD ./pkg/sentry/SENTRY_CONTEXT_VERSION
$CMD ./pkg/sentry/SENTRY_PRODUCTION
