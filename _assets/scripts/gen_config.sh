#!/usr/bin/env bash
set -euo pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

# Settings & defaults
RPC_HOST="${RPC_HOST:-localhost}"
RPC_PORT="${RPC_PORT:-8545}"
LISTEN_PORT="${LISTEN_PORT:-30303}"
API_MODULES="${API_MODULES:-eth,web3,admin}"
MAX_PEERS="${MAX_PEERS:-50}"
DAYS_KEPT="${DAYS_KEPT-30}"
FLEET_NAME="${FLEET_NAME:-status.prod}"
REGISTER_TOPIC="${REGISTER_TOPIC:-whispermail}"
MAIL_PASSWORD="${MAIL_PASSWORD:-status-offline-inbox}"
DATA_PATH="${DATA_PATH:-/var/tmp/status-go-mail}"
CONFIG_PATH="${CONFIG_PATH:-${DATA_PATH}/config.json}"

if ! [[ -x $(command -v jq) ]]; then
  echo "Cannot generate config. jq utility is not installed." >&2
  exit 1
fi

if [[ -e "${CONFIG_PATH}" ]]; then
  echo "Config already exits: ${CONFIG_PATH}" >&2
  echo "Remove it to generate a new one." >&2
  exit 0
fi

# Assemble the filter for changing the config JSON
JQ_FILTER_ARRAY=(
  ".ListenAddr = \"0.0.0.0:${LISTEN_PORT}\""
  ".HTTPEnabled = true"
  ".HTTPHost = \"${RPC_HOST}\""
  ".HTTPPort= ${RPC_PORT}"
  ".MaxPeers = ${MAX_PEERS}"
  ".DataDir = \"${DATA_PATH}\""
  ".APIModules = \"${API_MODULES}\""
  ".RegisterTopics = [\"${REGISTER_TOPIC}\"]"
  ".WakuConfig.Enabled = true"
  ".WakuConfig.EnableMailServer = true"
  ".WakuConfig.DataDir = \"${DATA_PATH}/waku\""
  ".WakuConfig.MailServerPassword = \"${MAIL_PASSWORD}\""
  ".WakuConfig.MailServerDataRetention = ${DAYS_KEPT}"
)

JQ_FILTER=$(printf " | %s" "${JQ_FILTER_ARRAY[@]}")

# make sure config destination exists
mkdir -p "${DATA_PATH}"

echo "Generating config at: ${CONFIG_PATH}"

cat "${GIT_ROOT}/config/cli/fleet-${FLEET_NAME}.json" \
    | jq "${JQ_FILTER:3}" > "${CONFIG_PATH}"
