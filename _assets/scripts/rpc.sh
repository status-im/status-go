#!/usr/bin/env bash
set -euo pipefail

RPC_HOST="${RPC_HOST:-localhost}"
RPC_PORT="${RPC_PORT:-8545}"
RPC_URL="${RPC_URL:-http://${RPC_HOST}:${RPC_PORT}/}"

METHOD="$1"
shift
PARAMS=("$@")

if [[ -z "${METHOD}" ]]; then
    echo "No method specified!" >&2
    exit 1
fi
# Parameter expansion trick to avoid var unbound error.
if [[ -z "${PARAMS-}" ]]; then
    PARAMS_STR=''
else
    PARAMS_STR=$(printf '%s\",\"' "${PARAMS[@]}")
    PARAMS_STR="\"${PARAMS_STR%%\",\"}\""
fi

PAYLOAD="{
  \"id\": 1,
  \"jsonrpc\": \"2.0\",
  \"method\": \"${METHOD}\",
  \"params\": [${PARAMS_STR}]
}"

OUT=$(
    curl --fail --show-error --silent \
        -H "Content-type:application/json" \
        -X POST --data "${PAYLOAD}" \
        "${RPC_URL}"
)

echo "${OUT}" | jq .
