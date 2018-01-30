metrics
=======

Currently, we aim to provide three options to two ways of collecting metrics:
* using [`expvar`](https://golang.org/pkg/expvar/) package for mobile device metrics collection using [statusmonitor](https://github.com/status-im/statusmonitor),
* using [Prometheus](https://prometheus.io/) for nodes running locally or on servers.

## Enable metrics

Metrics can be enabled with `-stats` flag when running it from command line. `-stats.addr` can be used to change metrics server address and port.

TBD: how to enable them when running `status-go` as a library?

## Build tags

To select how metrics are collected, `status-go` must be compiled with proper build tags.

To enable `expvar` metrics, compile it with `metrics` build tag:
```
make statusgo BUILD_TAGS='metrics'
```

To enable Prometheus metrics, compile it with `metrics` and `prometheus` tags:
```
make statusgo BUILD_TAGS='metrics prometheus'
```

If no `metrics` tag is provided, metrics won't be collected or will be printed as `DEBUG` logs depending on implementation.

## Current metrics

Currently, we have defined the following metrics.

### Whisper

#### expvar

* `envelope_counter` -- number of envelopes,
* `envelope_new_counter` -- number of new envelopes (arrived for the first time, not cached before),
* `envelope_topic_counter` -- a map with envelopes counted per topic,
* `envelope_peer_counter` -- a map with envelopes counted by peer,
* `envelope_volume` -- volume of all envelopes (sum of each envelope's size).

#### Prometheus

* `envelope_counter` -- count envelopes with labels: `topic`, `source` (values are `peer` or `p2p`), `is_new`, `peer`,
* `envelope_volume` -- incremental sum of envelopes volume with labels: `topic`, `source` (values are `peer` or `p2p`), `is_new`, `peer`.

### Node and peers

#### expvar

* `node_info` -- ID of the current node,
* `node_peers` -- marshaled list of all peers remote addresses.

#### Prometheus

Not available.
