#!/usr/bin/env bash
set -euo pipefail

# Removes this module's generated files. Dependency snapshots retain their
# bundled generated sources because this module's go generate does not rebuild them.

DRY_RUN="${CLEANUP_GENERATED_FILES_DRY_RUN:-false}"
CMD="rm -rf"
if [[ "$DRY_RUN" == "true" ]]; then
    CMD="echo [DRY-RUN]"
fi

echo "Cleaning up generated files... (dry run: $DRY_RUN) "

# Ignoring vendor is required for Nix builds. third_party contains local module
# replacements whose generated files are part of the pinned source distribution.
find_project_files() {
    find . \( -path './vendor' -o -path './third_party' \) -prune -o "$@"
}

find_project_files -type d -name 'mock' -exec $CMD {} +
find_project_files -type f -name 'mock.go' -exec $CMD {} +
find_project_files -type f -name '*_mock_test.go' -exec $CMD {} +
find_project_files -type f -name '*.pb.go' -exec $CMD {} +
find_project_files -type f -name 'bindata.go' -exec $CMD {} +
find_project_files -type f -name 'migrations.go' -exec $CMD {} +

$CMD ./cmd/status-backend/server/endpoints.go
$CMD ./internal/protocol/messenger_handlers.go
$CMD ./pkg/version/VERSION
$CMD ./pkg/version/GIT_COMMIT
$CMD ./pkg/sentry/SENTRY_CONTEXT_NAME
$CMD ./pkg/sentry/SENTRY_CONTEXT_VERSION
$CMD ./pkg/sentry/SENTRY_PRODUCTION
