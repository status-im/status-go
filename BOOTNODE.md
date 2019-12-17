# Description

This document describes the two easiest ways to start a Status Bootnode:

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
This will generate the necessary config, define and then start the service.

For more details read the [README](_assets/systemd/bootnode/README.md).
