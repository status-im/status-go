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
* `make config.json` - Creates `config.json` with your Public IP.

# Settings

All settings are passed through environment variables:

* `PUBLIC_IP` - Your IP visible from the internet and advertised by the Mailserver.
* `LISTEN_PORT` - Mailserver TCP & UDP port, by default it's `30303` but you might want to use `443`.
* `RPC_PORT` - Control port making it possible to use the [JSON-RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* `API_MODULES` - API modules to be made available via the `RPC_PORT`.
* `DATA_PATH` - Location of Mailserver storage and keys. (Default: `/var/tmp/status-go-mail`)
* `CONTAINER_NAME` - Name of the container that will be created.
* `REGISTER_TOPIC` - Mynamic mailserver discovery topic. (Default: `whispermail`)
* `MAIL_PASSWORD` - Basic HTTP auth password for mailserver. (Default: `status-offline-inbox`)
* `LOG_LEVEL` - Set level of log messages to show. (`ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`)

# Known Issues

If the discovery of your Public IP does not work please simply export the `PUBLIC_IP` env variable.
You can also set `LISTEN_PORT` to something else to avoid firewall issues.
```bash
$ export PUBLIC_IP=1.2.3.4
$ export LISTEN_PORT=443
$ make
docker-compose up -d
Creating status-go-mailserver ... done
CONTAINER ID        NAMES                  STATUS              PORTS
bee56564926d        status-go-mailserver   Up 6 minutes        8080/tcp, 127.0.0.1:8545->8545/tcp, 30303-30304/udp, 0.0.0.0:30303->30303/tcp
Your mailserver is listening on: 1.2.3.4:443
Make sure that address and port are available from the internet!
Your enode address is:
enode://dccd2f3c1df42c23af6672df28f287893ab70a5d45668637576a759b6db10b83e83fc02598f36c80ac094fbf8621419153cfe539f56d278ab099da21800f880c@127.0.0.1:30303
```
