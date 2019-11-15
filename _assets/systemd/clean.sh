#!/usr/bin/env bash

# Settings & defaults
export SERVICE_NAME="${SERVICE_NAME:-statusd}"

# stop before removing
systemctl --user stop "${SERVICE_NAME}"
systemctl --user disable "${SERVICE_NAME}"

# remove the service definition file
rm -f "${HOME}/.config/systemd/user/${SERVICE_NAME}.service"

# make systemd forget about it
systemctl --user daemon-reload
systemctl --user reset-failed
