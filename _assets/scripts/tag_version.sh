#!/usr/bin/env bash

set -euo pipefail

source _assets/scripts/commit_check.sh

get_latest_tag() {
    # Get the latest tag on develop
    latest_tag=$(git describe --tags --abbrev=0 develop)
    echo "$latest_tag"
}

bump_version() {
    local tag=$1
    local is_breaking_change=$2
    IFS='.' read -r major minor patch <<< "$tag"

    # Bump the version based on the type of change
    if [[ "$is_breaking_change" = true ]]; then
        ((minor++))
    else
        ((patch++))
    fi

    new_version="$major.$minor.$patch"
    echo "$new_version"
}

calculate_new_version() {
    # Get the latest tag
    latest_tag=$(get_latest_tag)

    echo "calculating new tag from $latest_tag and $1" >&2

    # Parse commits to determine if there are breaking changes
    is_breaking_change=$(parse_commits "$latest_tag" "$1")

    # Bump version accordingly
    echo "$(bump_version "$latest_tag" "$is_breaking_change")"
  }


main() {
    new_version=$(calculate_new_version "$1")
    echo "calculated new version: $new_version" >&2

    git tag -a "$new_version" "$1" -m "release $new_version"
}

target_commit=${1:-HEAD}

main "$target_commit"
