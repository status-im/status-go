#!/usr/bin/env bash
set -euo pipefail

GITHUB_ORG=$1
GITHUB_REPO=$2
SMART_CONTRACT_DIR=$3
SMART_CONTRACT_FILENAME=$4
PRIVATE_KEY=$5
SENDER_ADDRESS=$6

export GIT_HTTP_LOW_SPEED_LIMIT=1000
export GIT_HTTP_LOW_SPEED_TIME=30
# FIXME: hack fix for github asking auth on public resources
export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0=http.version
export GIT_CONFIG_VALUE_0=HTTP/1.1
export GIT_TERMINAL_PROMPT=0

retry() {
  local max=$1; shift
  local n
  for ((n=1; n<=max; n++)); do
    "$@" && return 0
    [[ $n -eq $max ]] && return 1
    sleep $((n * 5))
  done
}

clone_repo() {
  rm -rf "$GITHUB_REPO"
  git clone "https://github.com/$GITHUB_ORG/$GITHUB_REPO"
}

cd /app
retry 3 clone_repo
cd "$GITHUB_REPO"
git submodule deinit --force .
retry 3 git submodule update --init --recursive

forge build
forge script "$SMART_CONTRACT_DIR/$SMART_CONTRACT_FILENAME" \
  --fork-url="$ETH_RPC_URL" --private-key="$PRIVATE_KEY" \
  --sender="$SENDER_ADDRESS" --broadcast
