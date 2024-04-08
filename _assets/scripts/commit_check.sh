#!/usr/bin/env bash

set -euo pipefail

parse_commits() {
    start_commit=${1:-origin/develop}
    end_commit=${2:-HEAD}
    is_breaking_change=false

    echo "checking commits between: $start_commit $end_commit" >&2
    # Run the loop in the current shell using process substitution
    while IFS= read -r message || [ -n "$message" ]; do
        # Check if commit message follows conventional commits format
        if [[ $message =~ ^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\(.*\))?(\_|!):.*$ ]]; then
            # Check for breaking changes
            if [[ ${BASH_REMATCH[3]} == *'!'* ]]; then
                is_breaking_change=true
                break
            fi
        else
            echo "Commit message \"$message\" is not well-formed. Aborting merge. We use https://www.conventionalcommits.org/en/v1.0.0/ but with _ for non-breaking changes"
            # Uncomment the line below if you want to exit on an invalid commit message
            exit 1
        fi
    done < <(git log --format=%B "$start_commit".."$end_commit" | sed '/^\s*$/d')

    echo "$is_breaking_change"
}

parse_commits
