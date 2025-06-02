#!/usr/bin/env bash

set -euo pipefail

source _assets/scripts/colors.sh

check_migration_order() {
  local prev_migration=""
  for file in "$@"; do
    current_migration=$(basename "$file")

    if [[ ! -z "$prev_migration" && "$current_migration" < "$prev_migration" ]]; then
      echo -e "${YLW}migration ${RST}${current_migration} ${YLW}is not in order with ${RST}${prev_migration}"
      echo -e "${YLW}Error: Migration files are out of order. Please ensure migrations are added in chronological order."
      exit 1
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

git checkout ${BASE_COMMIT}
git pull origin ${BASE_BRANCH}
git checkout -

for MIGRATION_DIR in ${MIGRATION_DIRS[@]}; do
  echo -e "${GRN}Checking migrations:${RST} ${MIGRATION_DIR}"

  base_files=$(git ls-tree -r --name-only ${BASE_COMMIT} ${MIGRATION_DIR}/*.sql | sort)
  new_files=$(git diff --name-only ${BASE_COMMIT} ${MIGRATION_DIR}/*.sql | sort)
  all_files=$(echo -e "$base_files\n$new_files")

  check_migration_order $all_files
done

exit 0
