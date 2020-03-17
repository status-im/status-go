# Description

This package configures Prometheus metrics for the node.

# Technical Details

We use a trick to combine our metrics with Geth ones.

The `NewMetricsServer()` function in [`metrics.go`](./metrics.go) calls our own `Handler()` function which in turn calls two handlers:

* `promhttp.HandlerFor()` - Our own custom metrics from this package.
* `gethprom.Handler(reg)` - Geth metrics defined in [`metrics`](https://github.com/ethereum/go-ethereum/tree/master/metrics)

By calling both we can extend existing metrics.

# Metrics

We add a few extra metrics on top of the normal Geth ones in [`node/metrics.go`](./node/metrics.go):

* `p2p_peers_count` - Current numbers of peers split by name.
* `p2p_peers_absolute` - Absolute number of connected peers.
* `p2p_peers_max` - Maximum number of peers that can connect.

The `p2p_peers_count` metrics includes 3 labels:

* `type` - Set to `StatusIM` for mobile and `Statusd` for daemon.
* `version` - Version of `status-go`, always with the `v` prefix.
* `platform` - Host platform, like `android-arm64` or `darwin-arm64`

The way this data is acquired is using node names, which look like this:
```
StatusIM/vrelease-0.30.1-beta.2/android-arm/go1.11.5
Statusd/v0.34.0-beta.3/linux-amd64/go1.13.1
Geth/v1.9.9-stable-5aa131ca/linux-amd64/go1.13.3
```
This 4 segment format is standard for Ethereum as you can see on https://ethstats.net/.

We parse the names using `labelsFromNodeName()` from [`node/metrics.go`](./node/metrics.go).

# Links

* https://github.com/status-im/infra-misc/issues/26
* https://github.com/status-im/status-go/pull/1648
