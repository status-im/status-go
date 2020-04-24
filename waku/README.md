# `waku`

## Table of contents

- [What is Waku?](#what-is-waku)
- [What does this package do?](#what-does-this-package-do)
  - [waku.go](#wakugo)
  - [api.go](#apigo)
  - [config.go](#configgo)
  - [const.go](#constgo)
  - [envelope.go](#envelopego)
  - [events.go](#eventsgo)
  - [filter.go](#filtergo)
  - [handshake.go](#handshakego)
  - [mailserver.go](#mailservergo)
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

---

### `waku.go`

[`waku.go`](./waku.go) serves as the main entry point for the package and where the main `Waku{}` struct lives. Additionally the package's `init()` can be found in this file.

---

### `api.go`

[`api.go`](./api.go) is home to the `PublicWakuAPI{}` struct which provides the waku RPC service that can be used publicly without security implications.

`PublicWakuAPI{}` wraps the main `Waku{}`, making the `Waku{}` functionality suitable for external consumption.

#### Consumption

`PublicWakuAPI{}` is wrapped by `eth-node\bridge\geth.gethPublicWakuAPIWrapper{}`, which is initialised via `eth-node\bridge\geth.NewGethPublicWakuAPIWrapper()` and exposed via `gethWakuWrapper.PublicWakuAPI()` and is finally consumed by wider parts of the application.

---

### `config.go`

[`config.go`](./config.go) is home to the `Config{}` struct and the declaration of `DefaultConfig`.

`Config{}` is used to initialise the settings of an instantiated `Waku{}`. `waku.New()` creates a new instance of a `Waku{}` and takes a `Config{}` as a parameter, if nil is passed instead of an instance of `Config{}`, `DefaultConfig` is used. 

---

### `const.go`

[`const.go`](./const.go), originally a hangover from the [`go-ethereum` `whisperv6/doc.go` package file](https://github.com/ethereum/go-ethereum/blob/master/whisper/whisperv6/doc.go) later [refactored](https://github.com/status-im/status-go/pull/1950), is home to the package's constants.

---

### `envelope.go`

[`envelope.go`](./envelope.go) is home to the `Evelope{}` and `EnvelopeError{}` structs. `Envelope{}` is used as the data packet in which message data is sent through the Waku network.

`Envelope{}` is accessed via the initialisation function `NewEnvelope()`, which is exclusively consumed by `Message.Wrap()` that prepares a message to be sent via Waku. 

---

### `events.go`

[`events.go`](./events.go) handles data related to Waku events. This file contains string type `const`s that identify known Waku events.

Additionally, the file contains `EnvelopeEvent{}`, which serves as a representation of events created by envelopes. `EnvelopeEvent{}`s are initialised exclusively within the `waku` package.  

--- 

### `filter.go`

[`filter.go`](./filter.go) //TODO

---

### `handshake.go`

[`handshake.go`](./handshake.go) //TODO

### `mailserver.go`

[`mailserver.go`](./mailserver.go) //TODO

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
