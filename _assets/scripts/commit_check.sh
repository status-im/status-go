#!/usr/bin/env bash

set -euo pipefail

parse_commits() {

    BASE_BRANCH=${BASE_BRANCH:-develop}

    start_commit=${1:-origin/${BASE_BRANCH}}
    end_commit=${2:-HEAD}
    is_breaking_change=false
    exit_code=0

    echo "checking commits between: $start_commit $end_commit"
    # Run the loop in the current shell using process substitution
    while IFS= read -r message || [ -n "$message" ]; do
        # Check if commit message follows conventional commits format
        if [[ $message =~ ^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\(.*\))?(\_|!):.*$ ]]; then
            # Check for breaking changes
            if [[ ${BASH_REMATCH[3]} == *'!'* ]]; then
                is_breaking_change=true
            fi
        else
            echo "Commit message \"$message\" is not well-formed"
            exit_code=1
        fi
    done < <(git log --format=%s "$start_commit".."$end_commit")

    if [[ $exit_code -ne 0 ]]; then
        exit ${exit_code}
    fi

    echo "$is_breaking_change"
}

parse_commits "$@"
