# Status Mailserver

This folder contains setup for running your own Status Mailserver.
It uses:

* [Docker Compose](https://docs.docker.com/compose/) for managing the Status Mailserver container.
* [Makefile](https://www.gnu.org/software/make/) to simplify the process for the user.

# Requirements

This Makefile uses two tools: `jq` and `docker-compose`.

# Usage

To simply start a container run `make`, other commands include:

* `make start` - Starts the `status-go-mailserver` container.
* `make stop` - Stops the container.
* `make show` - Shows you current status of the container.
* `make logs` - Shows you logs of the container.
* `make config` - Creates `${DATA_PATH}/config.json` with your Public IP.
* `make enode` - Shows `enode://` address of the container.
* `make enode-qr` - Shows `enode://` address using a QR code.

# Settings

All settings are passed through environment variables:

* `LISTEN_PORT` - Mailserver TCP & UDP port, by default it's `30303` but you might want to use `443`.
* `METRICS_PORT` - Port exposing metrics in [Prometheus](https://prometheus.io/docs/concepts/data_model/) format
* `RPC_PORT` - Control port making it possible to use the [JSON-RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* `API_MODULES` - API modules to be made available via the `RPC_PORT`.
* `DATA_PATH` - Location of Mailserver storage and keys. (default: `/var/tmp/status-go-mail`)
* `CONTAINER_IMG` - Name of the container image. (default: `statusteam/status-go`)
* `CONTAINER_TAG` - Tag of container image to use. (default: `v0.64.3`) 
* `CONTAINER_NAME` - Name of the container that will be created. (default: `status-go-mailserver`)
* `REGISTER_TOPIC` - Mynamic mailserver discovery topic. (default: `whispermail`)
* `MAIL_PASSWORD` - Basic HTTP auth password for mailserver. (default: `status-offline-inbox`)
* `LOG_LEVEL` - Set level of log messages to show. (valid: `ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`)

The generated configuration file end up under `${DATA_PATH}/config.json`.

# F.A.Q.

### Should I change the `MailServerPassword` in the `WhisperConfig` section for better security?

This password is only used to symmetrically secure history requests. It is not used to secure messages or authorize users in any way. Whisper protocol requires each messages to be encrypted so a client and mailserver needs to agree on some password.
