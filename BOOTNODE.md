# Description

A "Bootnode" is a match-maker for all kinds of nodes. For example our [Mailserver](/MAILSERVER.md). It helps a new node find other nodes it can communicate with. It bootstraps connections between them.

A Bootstrap node is a very static thing and should not change its IP. Their addresses are either hardcoded into the node binary or provided via flags or config files on process start.

# Service Ports

* `30301` UDP - Ethereum [Discovery port](https://github.com/ethereum/devp2p/wiki/Discovery-Overview), should __ALWAYS__ be public.
# Setup methods

This document describes the two alternative ways to start a Status Bootnode:

* [Docker Compose](https://docs.docker.com/compose/) - More self-contained and portable
* [Systemd Service](https://www.freedesktop.org/wiki/Software/systemd/) - More local and configurable

## Docker Compose

The simplest way is to just use:
```
make run-bootnode-docker
```
This will generate the necessary config, compose and then start the container.

For more details read the [README](_assets/compose/bootnode/README.md).

## Systemd Service

The other way is to run the `bootnode` under `systemd`:
```
make run-bootnode-systemd
```
This will generate the necessary config, define and then start a user service.
Use `sudo` if you want it to be a system service.

For more details read the [README](_assets/systemd/bootnode/README.md).
