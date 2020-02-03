# Status Bootnode

This folder contains setup for running your own Status Bootnode.
It uses [Systemd](https://www.freedesktop.org/wiki/Software/systemd/) for managing the Status Bootnode service.

The steps it takes are:

* Builds `bootnode`
* Generates & saves a private key
* Generates `systemd` service
* Starts the service

# Usage

To simply configure and start the service run `make`.

In order to manage the new `statusd` service you use other `Makefile` targets:

* `make info` - Info about service
* `make enode` - Get enode address
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
* `PUBLIC_IP` - Your IP visible from the internet and advertised by the Bootnode.
* `LISTEN_PORT` - Bootnode TCP & UDP port, by default it's `30301` but you might want to use `443`.
* `DATA_PATH` - Location of Bootnode storage and keys. (Default: `/var/tmp/status-go-boot`)
* `KEY_PATH` - Location of Bootnode private key file. (Default: `/var/tmp/status-go-boot/nodekey`)
* `LOG_LEVEL` - Set level of log messages to show. (Values:`0-9`, Default: `3`)`

# System Service

By default this `Makefile` configures the Bootnode as a [systemd user service](https://www.freedesktop.org/software/systemd/man/user@.service.html). This is done to simplify the proces and remove the need for `sudo`. The disadvantage of this solution is that the service is stopped when the user logs out.

In order to make your service a system service use `sudo make`.

# Known Issues

* `No journal files were opened due to insufficient permissions.` from `systemctl`
  - To see logs of a user systemd service you need to be a member of `systemd-journal` group.
  - Use: `bash usermod -a -G systemd-journal ${USER}`
