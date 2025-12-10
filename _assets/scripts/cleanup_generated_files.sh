#!/usr/bin/env bash
set -euo pipefail

# Removes all generated files. List of non-tracked generated files can be found in .gitignore.

DRY_RUN="${DRY_RUN:-false}"
CMD="rm -rf"
if [[ "$DRY_RUN" == "true" ]]; then
    CMD="echo [DRY-RUN]"
fi

echo "Cleaning up generated files... (dry run: $DRY_RUN) "

# Ignoring vendor directory is required for nix builds.

find . -path './vendor' -prune -o -type d -name 'mock' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name 'mock.go' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name '*_mock_test.go' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name '*.pb.go' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name 'bindata.go' -exec $CMD {} +
find . -path './vendor' -prune -o -type f -name 'migrations.go' -exec $CMD {} +

$CMD ./cmd/status-backend/server/endpoints.go
$CMD ./protocol/messenger_handlers.go
$CMD ./pkg/version/VERSION
$CMD ./pkg/version/GIT_COMMIT
$CMD ./pkg/sentry/SENTRY_CONTEXT_NAME
$CMD ./pkg/sentry/SENTRY_CONTEXT_VERSION
$CMD ./pkg/sentry/SENTRY_PRODUCTION
