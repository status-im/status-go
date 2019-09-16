# Status Mailserver

This folder contains setup for running your own Status Mailserver.
It uses [Systemd](https://www.freedesktop.org/wiki/Software/systemd/) for managing the Status Mailserver service.

The steps it takes are:

* Builds statusd
* Generates `statusd` config
* Generates `systemd` service
* Starts the service

# Usage

To simply configure and start the service run `./start.sh`.

In order to manage the new `statusd` service you use `systemctl` command:

* `systemctl --user start statusd` - Start the service
* `systemctl --user stop statusd` - Stop the service
* `systemctl --user status statusd` - Check service status
* `systemctl --user disable statusd` - Disable the service
* `journalctl --user-unit statusd` - Read the service logs

If you want to remove the service you can just remove its definition:
```
systemctl --user stop statusd
rm ~/.config/systemd/user/statusd.service
systemctl --user daemon-reload
```

# Settings

All settings are passed through environment variables:

* `SERVICE_NAME` - Name of the `systemd` service to be created. (Default: `statusd`)
* `PUBLIC_IP` - Your IP visible from the internet and advertised by the Mailserver.
* `LISTEN_PORT` - Mailserver TCP & UDP port, by default it's `30303` but you might want to use `443`.
* `RPC_PORT` - Control port making it possible to use the [JSON-RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* `API_MODULES` - API modules to be made available via the `RPC_PORT`.
* `DATA_PATH` - Location of Mailserver storage and keys. (Default: `/var/tmp/status-go-mail`)
* `REGISTER_TOPIC` - Mynamic mailserver discovery topic. (Default: `whispermail`)
* `MAIL_PASSWORD` - Basic HTTP auth password for mailserver. (Default: `status-offline-inbox`)
* `LOG_LEVEL` - Set level of log messages to show. (`ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`)

The generated configuration file end up under `${DATA_PATH}/config.json`.

# Known Issues

* `No journal files were opened due to insufficient permissions.` from `systemctl`
  - To see logs of a user systemd service you need to be a member of `systemd-journal` group.
  - Use: `bash usermod -a -G systemd-journal ${USER}`
