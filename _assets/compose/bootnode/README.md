# Status Bootnode

This folder contains setup for running your own Bootnode.
It uses:

* [Docker Compose](https://docs.docker.com/compose/) for managing the bootnode container.
* [Makefile](https://www.gnu.org/software/make/) to simplify the process for the user.

# Requirements

This Makefile uses two tools: `jq` and `docker-compose`.

# Usage

To simply start a container run `make`, other commands include:

* `make start` - Starts the `status-go-bootnode` container.
* `make stop` - Stops the container.
* `make show` - Shows you current status of the container.
* `make logs` - Shows you logs of the container.
* `make enode` - Shows `enode` address of bootnode.
* `make keys/nodekey` - Creates unique identity for bootnode.

# Settings

All settings are passed through environment variables:

* `LISTEN_PORT` - Bootnode UDP port, by default it's `30301` but you might want to use `443`.
* `RPC_PORT` - Control port making it possible to use the [JSON-RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* `API_MODULES` - API modules to be made available via the `RPC_PORT`.
* `CONTAINER_IMG` - Name of the container image. (default: `statusteam/status-go`)
* `CONTAINER_TAG` - Tag of container image to use. (default: `v0.64.3`)
* `CONTAINER_NAME` - Name of the container that will be created. (default: `status-go-bootnode`)
* `LOG_LEVEL` - Set level of log messages to show. (valid: `1=ERROR`, `2=WARN`, `3=INFO`, `4=DEBUG`, `5=TRACE`)
* `COMPOSE_UP_FLAGS` - Allows passing flags to `docker-compose up`
