#!/usr/bin/env bash
set -eu

pr_body=$(gh api "repos/status-im/status-go/pulls/$1" --jq '.body // ""')

if [[ "$pr_body" =~ https://github\.com/status-im/status-app/pull/([0-9]+) ]]; then
    companion_pr=${BASH_REMATCH[1]}
elif [[ "$pr_body" =~ status-im/status-app#([0-9]+) ]]; then
    companion_pr=${BASH_REMATCH[1]}
else
    echo "REASON=no_link_in_description"
    exit 1
fi

companion_json=$(gh api "repos/status-im/status-app/pulls/${companion_pr}")
companion_title=$(echo "$companion_json" | jq -r '.title' | tr '\n' ' ')
companion_sha=$(echo "$companion_json" | jq -r '.head.sha')

submodule_sha=$(gh api "repos/status-im/status-app/contents/vendor/status-go?ref=${companion_sha}" --jq '.sha')

if [[ "$submodule_sha" != "$2" ]]; then
    echo "COMPANION_PR_NUMBER=${companion_pr}"
    echo "COMPANION_PR_TITLE=${companion_title}"
    echo "COMPANION_PR_URL=https://github.com/status-im/status-app/pull/${companion_pr}"
    echo "REASON=sha_mismatch"
    exit 1
fi

echo "COMPANION_PR_NUMBER=${companion_pr}"
echo "COMPANION_PR_TITLE=${companion_title}"
echo "COMPANION_PR_URL=https://github.com/status-im/status-app/pull/${companion_pr}"
echo "VERIFIED=true"
