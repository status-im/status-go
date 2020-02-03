# Description

A poorly named "Mailserver" is essentially a Whisper node that stores message history in either a LevelDB or PostgreSQL database.

A Status app user can run their own Mailserver for faster message retrieval or additional security.

# Service Ports

* `30303` TCP - [DevP2P](https://github.com/ethereum/devp2p) wire protocol port. Should __ALWAYS__ be public.
* `8545` TCP - [JSON RPC](https://github.com/ethereum/wiki/wiki/json-rpc) management port. Should __NEVER__ be public.

# Setup methods

This document describes the two alternative ways to start a Status Mailserver:

* [Docker Compose](https://docs.docker.com/compose/) - More self-contained and portable
* [Systemd Service](https://www.freedesktop.org/wiki/Software/systemd/) - More local and configurable

## Docker Compose

The simplest way is to just use:
```
make run-mailserver-docker
```
This will generate the necessary config, compose and then start the container.

For more details read the [README](_assets/compose/mailserver/README.md).

## Systemd Service

The other way is to run the `mailserver` under `systemd`:
```
make run-mailserver-systemd
```
This will generate the necessary config, define and then start a user service.
Use `sudo` if you want it to be a system service.

For more details read the [README](_assets/systemd/mailserver/README.md).
