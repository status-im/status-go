#!/bin/bash

set -e
set -o pipefail

check_migration_order() {
  local prev_migration=""
  for file in "$@"; do
    current_migration=$(echo "$file" | cut -d'-' -f1)

    if [[ ! -z "$prev_migration" && "$current_migration" < "$prev_migration" ]]; then

      echo "migration ${current_migration} is not in order with ${prev_migration}"
      echo "Error: Migration files are out of order. Please ensure migrations are added in chronological order."
      exit 1
    fi

    prev_migration="$current_migration"
  done
}

git fetch origin develop
committed_files=$(git ls-tree -r --name-only HEAD protocol/migrations/sqlite/*.sql | sort)
staged_files=$(git diff --name-only origin/develop protocol/migrations/sqlite/*.sql | sort)

all_files=$(echo -e "$committed_files\n$staged_files")

check_migration_order $all_files

exit 0
