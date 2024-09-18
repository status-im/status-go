#!/usr/bin/env bash

set -euo pipefail

source _assets/scripts/parse_commits.sh
source _assets/scripts/colors.sh

get_latest_tag() {
    # Get the latest tag on develop
    latest_tag=$(git describe --tags --abbrev=0 develop)
    echo "$latest_tag"
}

bump_version() {
    local tag=$1
    local is_breaking_change=$2
    IFS='v.' read -r _ major minor patch <<< "$tag"

    # Bump the version based on the type of change
    if [[ "$is_breaking_change" = true ]]; then
        ((major++))
        ((minor=0))
        ((patch=0))
    else
        ((minor++))
        ((patch=0))
    fi

    new_version="$major.$minor.$patch"
    echo "v$new_version"
}

calculate_new_version() {
    target_commit=$1
    latest_tag=$2

    # Parse commits to determine if there are breaking changes
    output=$(parse_commits "$latest_tag" "$target_commit")
    exit_code=$?
    echo "$output" | sed '$d' >&2 # Skip the last line, it contains the breaking change flag

    is_breaking_change=$(echo "$output" | tail -n 1)

    if [[ $is_breaking_change == 'true' ]]; then
      echo -e "${YLW}Breaking change detected${RST}" >&2
    fi

    if [[ $exit_code -ne 0 && $is_breaking_change != true ]]; then
        echo -e "${YLW}Some commits are ill-formed, can not to auto-calculate new version${RST}" >&2
        read -p "Any of the commits above have a breaking change? (y/n): " yn
        case $yn in
            [Yy]* ) is_breaking_change=true;;
            [Nn]* ) is_breaking_change=false;;
            * ) echo "Please answer yes or no."; exit 1;;
        esac
    fi

    # Bump version accordingly
    bump_version "$latest_tag" "$is_breaking_change"
}

latest_tag=$(get_latest_tag)
echo -e "${GRN}Latest tag found:${RST} $latest_tag" >&2

target_commit=${1:-HEAD}
echo -e "${GRN}Calculating new version for:${RST} $target_commit" >&2

new_version=$(calculate_new_version "$target_commit" "$latest_tag")
echo -e "${GRN}Calculated new version:${RST} $new_version" >&2

git tag -a "$new_version" "$target_commit" -m "release $new_version"
