#!/usr/bin/env bash

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

mkdir -p "${HOME}/.config/systemd/user"

cat >"${HOME}/.config/systemd/user/${SERVICE_NAME}.service" << EOF
[Unit]
Description=Status.im Mailserver Service

[Service]
Type=notify
Restart=on-failure
WatchdogSec=60s
WorkingDirectory=${DATA_PATH}
ExecStart=${GIT_ROOT}/build/bin/statusd \\
    -log "${LOG_LEVEL}" \\
    -log-without-color \\
    -dir "${DATA_PATH}" \\
    -c "./config.json"

[Install]
WantedBy=default.target
EOF
