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

* `PUBLIC_IP` - Your IP visible from the internet and advertised by the Bootnode.
* `LISTEN_PORT` - Bootnode port, by default it's `30303` but you might want to use `443`.
* `RPC_PORT` - Control port making it possible to use the [JSON-RPC API](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* `API_MODULES` - API modules to be made available via the `RPC_PORT`.
* `CONTAINER_NAME` - Name of the container that will be created.
* `LOG_LEVEL` - Set level of log messages to show. (__default:__ `1=ERROR`, `2=WARN`, `3=INFO`, `4=DEBUG`, `5=TRACE`)

# Known Issues

If the discovery of your Public IP does not work please simply export the `PUBLIC_IP` env variable.
You can also set `LISTEN_PORT` to something else to avoid firewall issues.
```bash
$ export PUBLIC_IP=1.2.3.4
$ export LISTEN_PORT=443
$ make
Created key for Bootnode: keys/nodekey
docker-compose up -d
Creating status-go-bootnode ... done
CONTAINER ID        NAMES                STATUS                  PORTS
036ca55423c2        status-go-bootnode   Up Less than a second   0.0.0.0:30303->30303/tcp
Your bootnode is listening on: 1.2.3.4:30303
Make sure that address and port are available from the internet!
Your enode address is:
enode://2654f0ee4f021cecbfc84d56e7749daa01250f5fd3828989e482b36f9ca13981c83d4c28ad647e4cfec26efa6b8d230075d17fc9c0eb127a7210bd35d27a67a9@1.2.3.4:30303
```
