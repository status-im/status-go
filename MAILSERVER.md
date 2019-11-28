# Description

This document describes the two easiest ways to start a Status Mailserver:

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
This will generate the necessary config, define and then start the service.

For more details read the [README](_assets/systemd/mailserver/README.md).
