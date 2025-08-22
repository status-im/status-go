# Description

This package configures Prometheus metrics for the node.

# Technical Details

We use a trick to combine our metrics with Geth ones.

The `NewMetricsServer()` function in [`metrics.go`](metrics.go) creates a new server 
which allows to expose multiple prometheus handler (e.g. status and waku).