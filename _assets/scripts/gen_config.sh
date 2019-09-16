#!/usr/bin/env bash

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

# Settings & defaults
RPC_PORT="${RPC_PORT:-8545}"
API_MODULES="${API_MODULES:-eth,net,web3,admin}"
FLEET_NAME="${FLEET_NAME:-eth.beta}"
REGISTER_TOPIC="${REGISTER_TOPIC:-whispermail}"
MAIL_PASSWORD="${MAIL_PASSWORD:-status-offline-inbox}"
DATA_PATH="${DATA_PATH:-/var/tmp/status-go-mail}"
CONFIG_PATH="${CONFIG_PATH:-${DATA_PATH}/config.json}"

if [[ -e "${CONFIG_PATH}" ]]; then
  echo "Config already exits. Remove it to generate a new one."
  exit 0
fi

# Necessary to make mailserver available publicly
export PUBLIC_IP=$(curl -s https://ipecho.net/plain)

# Assemble the filter for changing the config JSON
JQ_FILTER_ARRAY=(
  ".AdvertiseAddr = \"${PUBLIC_IP}\""
  ".HTTPEnabled = true"
  ".HTTPHost = \"0.0.0.0\""
  ".HTTPPort= ${RPC_PORT}"
  ".APIModules = \"${API_MODULES}\""
  ".RegisterTopics = [\"${REGISTER_TOPIC}\"]"
  ".WhisperConfig.Enabled = true"
  ".WhisperConfig.EnableMailServer = true"
  ".WhisperConfig.LightClient = false"
  ".WhisperConfig.MailServerPassword = \"${MAIL_PASSWORD}\""
)

JQ_FILTER=$(printf " | %s" "${JQ_FILTER_ARRAY[@]}")

echo "Generating config at: ${CONFIG_PATH}"

cat "${GIT_ROOT}/config/cli/fleet-${FLEET_NAME}.json" \
    | jq "${JQ_FILTER:3}" > "${CONFIG_PATH}"
