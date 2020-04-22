# `waku`

## Table of contents

- [What is Waku?](#what-is-waku)
- [What does this package do?](#what-does-this-package-do)
  - [waku.go](#wakugo)
  - [api.go](#apigo)
  - [config.go](#configgo)
  - [doc.go](#docgo)
  - [envelope.go](#envelopego)
  - [events.go](#eventsgo)
  - [filter.go](#filtergo)
  - [handshake.go](#handshakego)
  - [mailserver_response.go](#mailserver_responsego)
  - [message.go](#messagego)
  - [metrics.go](#metricsgo)
  - [peer.go](#peergo)
  - [rate_limiter.go](#rate_limitergo)
  - [topic.go](#topicgo)

## What is Waku?

Waku is a communication protocol for sending messages between Dapps. Waku is a fork of the [Ethereum Whisper subprotocol](https://github.com/ethereum/wiki/wiki/Whisper), although not directly compatible with Whisper, both Waku and Whisper subprotocols can communicate [via bridging](https://github.com/vacp2p/specs/blob/master/specs/waku/waku-1.md#backwards-compatibility).

Waku was [created to solve scaling issues with Whisper](https://discuss.status.im/t/fixing-whisper-for-great-profit/1419) and [currently diverges](https://github.com/vacp2p/specs/blob/master/specs/waku/waku-1.md#differences-between-shh6-and-waku1) from Whisper in the following ways:

- RLPx subprotocol is changed from `shh/6` to `waku/1`.
- Light node capability is added.
- Optional rate limiting is added.
- Status packet has following additional parameters: light-node, confirmations-enabled and rate-limits
- Mail Server and Mail Client functionality is now part of the specification.
- P2P Message packet contains a list of envelopes instead of a single envelope.

## What does this package do? 

The basic function of this package is to implement the [waku specifications](https://github.com/vacp2p/specs/blob/master/specs/waku/waku-1.md), and provide the `status-go` binary with the ability to send and receive messages via Waku.

### `waku.go`

[`waku.go`](./waku.go) serves as the main entry point for the package and where the main `Waku{}` struct lives. Additionally the package's `init()` can be found in this file.

### `api.go`

[`api.go`](./api.go) //TODO

### `config.go`

[`config.go`](./config.go) //TODO

### `doc.go`

[`doc.go`](./doc.go) //TODO

### `envelope.go`

[`envelope.go`](./envelope.go) //TODO

### `events.go`

[`events.go`](./events.go) //TODO

### `filter.go`

[`filter.go`](./filter.go) //TODO

### `handshake.go`

[`handshake.go`](./handshake.go) //TODO

### `mailserver_response.go`

[`mailserver_response.go`](./mailserver_response.go) //TODO

### `message.go`

[`message.go`](./message.go) //TODO

### `metrics.go`

[`metrics.go`](./metrics.go) //TODO

### `peer.go`

[`peer.go`](./peer.go) //TODO

### `rate_limiter.go`

[`rate_limiter.go`](./rate_limiter.go) //TODO

### `topic.go`

[`topic.go`](./topic.go) //TODO
