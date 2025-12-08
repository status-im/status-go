#!/usr/bin/env bash

set -euo pipefail

source _assets/scripts/colors.sh

# Track whether any validation failed to report all issues before exiting
FAILED=0

# Verify filename starts with exactly 10 digits before the first underscore
validate_filename_regex() {
  local fname="$1"
  local base
  base=$(basename "$fname")

  if [[ ! "$base" =~ ^[0-9]{10}_ ]]; then
    echo -e "${YLW}Error:${RST} migration '${base}' must start with a seconds-precision timestamp followed by an underscore."
    return 1
  fi
  return 0
}

check_migration_order() {
  local prev_migration=""

  for file in "$@"; do
    local current_migration
    current_migration=$(basename "$file")

    echo "Checking order: ${current_migration} against ${prev_migration}"

    # String-based order check (filenames sorted lexicographically)
    if [[ -n "$prev_migration" && "$current_migration" < "$prev_migration" ]]; then
      echo -e "${YLW}Error:${RST}migration ${current_migration} ${YLW}is not in chronological order with ${RST}${prev_migration}"
      # GitHub annotation on the problematic current file
      echo "::error file=${file},line=1,col=1::Migration order incorrect: '${current_migration}' should come after '${prev_migration}'."
      FAILED=1
      # continue checking other files
    fi

    prev_migration="$current_migration"
  done
}

BASE_BRANCH=${BASE_BRANCH:-develop}
BASE_COMMIT=${1:-origin/${BASE_BRANCH}}

MIGRATION_DIRS=( \
  "protocol/migrations/sqlite" \
  "appdatabase/migrations/sql" \
  "protocol/encryption/migrations/sqlite" \
  "walletdatabase/migrations/sql" \
)

# Update base ref locally to ensure comparisons are accurate
echo -e "${GRN}Checking out${RST} ${BASE_COMMIT} to verify against ${BASE_BRANCH}"
git checkout ${BASE_COMMIT}
git pull origin ${BASE_BRANCH}
git checkout -

for MIGRATION_DIR in "${MIGRATION_DIRS[@]}"; do
  echo -e "${GRN}Checking migrations:${RST} ${MIGRATION_DIR}"

  # Compute the common ancestor (merge-base) between BASE_COMMIT and HEAD
  MB=$(git merge-base "${BASE_COMMIT}" HEAD) || { echo "no merge-base"; exit 1; }

  # Files present in BASE_COMMIT
  base_files=$(git ls-tree -r --name-only "${BASE_COMMIT}" -- "${MIGRATION_DIR}/*.sql" | sort)

  # Files changed on this branch since the merge-base
  new_files=$(git diff --name-only "${BASE_COMMIT}...HEAD" -- "${MIGRATION_DIR}/*.sql" | sort)

  # Combine lists
  all_files=$(echo -e "$base_files\n$new_files")

  # Regex validation: ONLY verify newly added/changed files match ^[0-9]{10}_ prefix
  if [[ -n "$new_files" ]]; then
    while IFS= read -r nf; do
      [[ -z "$nf" ]] && continue
      if ! validate_filename_regex "$(basename "$nf")"; then
        # Provide GitHub annotation with the file path
        echo "::error file=${nf},line=1,col=1::Migration filename must start with exactly 10 digits (Unix seconds) followed by an underscore. Example: '1725978456_my_migration.up.sql'"
        FAILED=1
        # continue checking others
      fi
    done <<< "$new_files"
  fi

  # Iterate in filename order and ensure lexicographic ordering
  check_migration_order $all_files
done

# Exit with failure if any issues were detected (so the job fails and annotations show up)
if [[ "$FAILED" -ne 0 ]]; then
  exit 1
fi
exit 0
