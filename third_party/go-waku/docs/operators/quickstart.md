# Quickstart: running a go-waku node

This guide explains how to build and run a go-waku node
for the most common use cases.
For a more advanced configuration see our [configuration guides](./how-to/configure.md)

## 1. Build

[Build the go-waku node](./how-to/build.md)
or download a precompiled binary from our [releases page](https://github.com/waku-org/go-waku/releases).
<!-- Docker images are published to [wakuorg/go-waku](https://hub.docker.com/r/wakuorg/go-waku/tags) on DockerHub. -->
<!-- TODO: more advanced explanation on finding and using docker images -->

## 2. Run

[Run the go-waku node](./how-to/run.md) using a default or common configuration
or [configure](./how-to/configure.md) the node for more advanced use cases.

[Connect](./how-to/connect.md) the go-waku node to other peers to start communicating.

## 3. Interact

A running go-waku node can be interacted with using the [Waku v2 JSON RPC API](https://rfc.vac.dev/spec/16/).

> **Note:** Private and Admin API functionality are disabled by default.
To configure a go-waku node with these enabled,
use the `--rpc-admin:true` and `--rpc-private:true` CLI options.
