# Status Mailserver

This folder contains setup for running your own Status Mailserver.
It uses [Systemd](https://www.freedesktop.org/wiki/Software/systemd/) for managing the Status Mailserver service.

The steps it takes are:

* Builds statusd
* Generates `statusd` config
* Generates `systemd` service
* Starts the service

# Usage

To simply configure and start the service run `make`.

In order to manage the new `statusd` service you use other `Makefile` targets:

* `make info` - Info about service
* `make enode` - Get enode address
* `make enode-qr` - Get enode address as QR code
* `make start` - Start the service
* `make stop` - Stop the service
* `make status` - Check service status
* `make enable` - Enable the service
* `make disable` - Disable the service
* `make logs` - Read the service logs
* `make clean` - Stop service and remove it

All the above commands are just wrappers around the [`systemctl`](http://man7.org/linux/man-pages/man1/systemctl.1.html) and [`journalctl`](http://man7.org/linux/man-pages/man1/journalctl.1.html) commands.

# Settings

All settings are passed through environment variables:

* `SERVICE_NAME` - Name of the `systemd` service to be created. (Default: `statusd`)
* `LISTEN_PORT` - Mailserver TCP & UDP port, by default it's `30303` but you might want to use `443`.
* `METRICS_PORT` - Port exposing metrics in [Prometheus](https://prometheus.io/docs/concepts/data_model/) format
* `RPC_PORT` - Control port making it possible to use the [JSON-RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* `API_MODULES` - API modules to be made available via the `RPC_PORT`.
* `DATA_PATH` - Location of Mailserver storage and keys. (Default: `/var/tmp/status-go-mail`)
* `REGISTER_TOPIC` - Mynamic Mailserver discovery topic. (Default: `whispermail`)
* `MAIL_PASSWORD` - Basic HTTP auth password for Mailserver. (Default: `status-offline-inbox`)
* `LOG_LEVEL` - Set level of log messages to show. (`ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`)

The generated configuration file end up under `${DATA_PATH}/config.json`.

# System Service

By default this `Makefile` configures the Mailserver as a [systemd user service](https://www.freedesktop.org/software/systemd/man/user@.service.html). This is done to simplify the proces and remove the need for `sudo`. The disadvantage of this solution is that the service is stopped when the user logs out.

In order to make your service a system service use `sudo make`.

# Known Issues

* `No journal files were opened due to insufficient permissions.` from `systemctl`
  - To see logs of a user systemd service you need to be a member of `systemd-journal` group.
  - Use: `bash usermod -a -G systemd-journal ${USER}`
