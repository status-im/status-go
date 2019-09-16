#!/usr/bin/env bash

RED=$(tput -Txterm setaf 1)
GRN=$(tput -Txterm setaf 2)
YLW=$(tput -Txterm setaf 3)
RST=$(tput -Txterm sgr0)
BLD=$(tput bold)

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

# Settings & defaults
export SERVICE_NAME="${SERVICE_NAME:-statusd}"
export LOG_LEVEL="${LOG_LEVEL:-INFO}"
export LISTEN_PORT="${LISTEN_PORT:-30303}"
export DATA_PATH="${DATA_PATH:-/var/tmp/status-go-mail}"
# Necessary to make mailserver available publicly
export PUBLIC_IP=$(curl -s https://ipecho.net/plain)

function show_info() {
  systemctl --user status --no-pager ${SERVICE_NAME}
  echo
  # just nice to show at the end
  ENODE=$("${GIT_ROOT}/_assets/scripts/get_enode.sh")

  echo "* ${GRN}Your mailserver is listening on:${RST} ${BLD}${PUBLIC_IP}:${LISTEN_PORT}${RST}"
  echo "* ${YLW}Make sure that IP and TCP port are available from the internet!${RST}"
  echo -e "${GRN}Your enode address is:${RST}\n${ENODE}"
  exit 0
}

if ! [[ -x "$(command -v systemctl)" ]]; then
  echo "${RED}Your system does not have systemd!${RST}"
  exit 1
fi

# if the service is already up just show some info
if systemctl --user is-active --quiet ${SERVICE_NAME}; then
  echo "${YLW}Service already started!${RST}"
  show_info
fi

# if the service has failed just show the status
if systemctl --user is-failed --quiet ${SERVICE_NAME}; then
  echo "${RED}Service has failed!${RST}"
  systemctl --user status --no-pager ${SERVICE_NAME}
  exit 1
fi

# Build the statusd binary
# TODO possibly download it in the future
if [[ ! -x "${GIT_ROOT}/build/bin/statusd" ]]; then
    echo "* ${BLD}Building mailserver binary...${RST}"
    cd "${GIT_ROOT}" && make statusgo
fi

echo "* ${BLD}Generating '${SERVICE_NAME}' config...${RST}"
"${GIT_ROOT}/_assets/scripts/gen_config.sh"

echo "* ${BLD}Generating '${SERVICE_NAME}' service...${RST}"
"${GIT_ROOT}/_assets/systemd/gen_service.sh"

echo "* ${BLD}Enabling '${SERVICE_NAME}' service...${RST}"
systemctl --user enable ${SERVICE_NAME}

echo "* ${BLD}Starting '${SERVICE_NAME}' service...${RST}"
systemctl --user restart ${SERVICE_NAME}

show_info
