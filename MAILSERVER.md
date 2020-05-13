# Description

A poorly named "Mailserver" is essentially a Whisper node that stores message history in either a LevelDB or PostgreSQL database.

A Status app user can run their own Mailserver for faster message retrieval or additional security.

# Service Ports

* `30303` TCP/UDP - [DevP2P](https://github.com/ethereum/devp2p) wire protocol port. Must __ALWAYS__ be public.
* `8545` TCP - [JSON RPC](https://github.com/ethereum/wiki/wiki/json-rpc) management port. Must __NEVER__ be public.
* `9090` TCP - [Prometheus](https://prometheus.io/docs/concepts/data_model/) metrics port. Should not be public.

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

# Service Healthcheck

There's two simple ways to verify your Mailserver is up and running.

## Query Metrics

By making an HTTP request to the metrics port(`9090` by default) you can check if you Mailserver is receiving envelopes:
```sh
 $ curl -s localhost:9090/metrics | grep '^whisper_envelopes_received_total'
whisper_envelopes_received_total 123
```

## JSON RPC Calls

The JSON RPC port (`8545` by default) allows you to manage your node.
You can list connected peers by doing:
```sh
 $ export DATA='{"jsonrpc":"2.0","method":"admin_peers", "params": [], "id":1}'
 $ curl -s -H 'content-type: application/json' -d $DATA localhost:8545 \
     | jq -r '.result[].network.remoteAddress'
34.68.132.118:30305
134.209.136.123:30305
178.128.141.249:443
```
