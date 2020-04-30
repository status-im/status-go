# `waku`

## Table of contents

- [What is Waku?](#what-is-waku)
- [Waku versioning](#waku-versioning)
- [What does this package do?](#what-does-this-package-do)
- [Waku package files](#waku-package-files)

## What is Waku?

Waku is a communication protocol for sending messages between Dapps. Waku is a fork of the [Ethereum Whisper subprotocol](https://github.com/ethereum/wiki/wiki/Whisper), although not directly compatible with Whisper, both Waku and Whisper subprotocols can communicate [via bridging](https://github.com/vacp2p/specs/blob/master/specs/waku/waku-1.md#backwards-compatibility).

Waku was [created to solve scaling issues with Whisper](https://discuss.status.im/t/fixing-whisper-for-great-profit/1419) and [currently diverges](https://github.com/vacp2p/specs/blob/master/specs/waku/waku-1.md#differences-between-shh6-and-waku1) from Whisper in the following ways:

- RLPx subprotocol is changed from `shh/6` to `waku/1`.
- Light node capability is added.
- Optional rate limiting is added.
- Status packet has following additional parameters: light-node, confirmations-enabled and rate-limits
- Mail Server and Mail Client functionality is now part of the specification.
- P2P Message packet contains a list of envelopes instead of a single envelope.

## Waku versioning



## What does this package do? 

The basic function of this package is to implement the [waku specifications](https://github.com/vacp2p/specs/blob/master/specs/waku/waku-1.md), and provide the `status-go` binary with the ability to send and receive messages via Waku.

## Waku package files

  - [waku.go](#wakugo)
  - [api.go](#apigo)
  - [config.go](#configgo)
  - [mailserver.go](#mailservergo)
  - [common](#common)
    - [bloomfilter.go](#bloomfiltergo)
    - [const.go](#constgo)
    - [envelope.go](#envelopego)
    - [errors.go](#errorsgo)
    - [events.go](#eventsgo)
    - [filter.go](#filtergo)
    - [helpers.go](#helpersgo)
    - [message.go](#messagego)
    - [metrics.go](#metricsgo)
    - [protocol.go](#protocolgo)
    - [rate_limiter.go](#rate_limitergo)
    - [topic.go](#topicgo)
  - [Versioned](#versioned)
     - [const.go](#version-constgo)
     - [init.go](#version-initgo)
     - [message.go](#version-messagego)
     - [peer.go](#version-peergo)
     - [status_options.go](#version-status_optionsgo)

## Root

### `waku.go`

[`waku.go`](./waku.go) serves as the main entry point for the package and where the main `Waku{}` struct lives. Additionally the package's `init()` can be found in this file.

---

### `api.go`

[`api.go`](./api.go) is home to the `PublicWakuAPI{}` struct which provides the waku RPC service that can be used publicly without security implications.

`PublicWakuAPI{}` wraps the main `Waku{}`, making the `Waku{}` functionality suitable for external consumption.

---

#### Consumption

`PublicWakuAPI{}` is wrapped by `eth-node\bridge\geth.gethPublicWakuAPIWrapper{}`, which is initialised via `eth-node\bridge\geth.NewGethPublicWakuAPIWrapper()` and exposed via `gethWakuWrapper.PublicWakuAPI()` and is finally consumed by wider parts of the application.

---

### `config.go`

[`config.go`](./config.go) is home to the `Config{}` struct and the declaration of `DefaultConfig`.

`Config{}` is used to initialise the settings of an instantiated `Waku{}`. `waku.New()` creates a new instance of a `Waku{}` and takes a `Config{}` as a parameter, if nil is passed instead of an instance of `Config{}`, `DefaultConfig` is used. 

---

### `mailserver.go`

[`mailserver.go`](./mailserver.go) //TODO

---

## Common

### `bloomfilter.go`

[`bloomfilter.go`](./common/bloomfilter.go) //TODO

---

### `const.go`

[`const.go`](./common/const.go), originally a hangover from the [`go-ethereum` `whisperv6/doc.go` package file](https://github.com/ethereum/go-ethereum/blob/master/whisper/whisperv6/doc.go) later [refactored](https://github.com/status-im/status-go/pull/1950), is home to the package's constants.

---

### `envelope.go`

[`envelope.go`](./common/envelope.go) is home to the `Evelope{}` and `EnvelopeError{}` structs. `Envelope{}` is used as the data packet in which message data is sent through the Waku network.

`Envelope{}` is accessed via the initialisation function `NewEnvelope()`, which is exclusively consumed by `Message.Wrap()` that prepares a message to be sent via Waku. 

---

### `errors.go`

[`errors.go`](./common/errors.go) //TODO

---

### `events.go`

[`events.go`](./common/events.go) handles data related to Waku events. This file contains string type `const`s that identify known Waku events.

Additionally, the file contains `EnvelopeEvent{}`, which serves as a representation of events created by envelopes. `EnvelopeEvent{}`s are initialised exclusively within the `waku` package.  

--- 

### `filter.go`

[`filter.go`](./common/filter.go) is home to `Filter{}` which represents a waku filter.

#### Usage

A `status-go` node will install / register filters through RPC calls from a client (eg `status-react`). When the client installs a filter, it will specify 2 things:

1) An encryption key, example "`superSafeEncryptionKey`"
2) A 4 byte topic (`TopicType`), example "`0x1234`"

The node will install the filter `["superSafeEncryptionKey", 0x1234]` and will notify its peers of this event

When a node receives an envelope it will attempt to match the topics against the installed filters, and then try to decrypt the envelope if the topic matches.

For example, if a node receives an envelope with topic `0x1234`, the node will try to use the installed filter key `superSafeEncryptionKey` to decrypt the message. On success the node passes the decrypted message to the client.

**Waku / Whisper divergence**

Whisper, will process all the installed filters that the node has, and build a `BloomFilter` from all the topics of each installed filter (i.e. `func ToBloomFilter(topics []TopicType) []byte { ... }`). When a peer receives this BloomFilter, it will match the topic on each envelope that they receive against the BloomFilter, if it matches, it will forward this to the peer.

Waku, by default, does not send a BloomFilter, instead sends the topic in a clear array of `[]TopicType`. This is an improvement on Whisper's usage as a BloomFilter may include false positives, which increase bandwidth usage. In contrast, clear topics are matched exactly and therefore don't create redundant bandwidth usage.

---

### `helpers.go`

[`helpers.go`](./common/helpers.go) //TODO

---

### `message.go`

[`message.go`](./common/message.go) //TODO

---

### `metrics.go`

[`metrics.go`](./common/metrics.go) //TODO

---

### `protocol.go`

[`protocol.go`](./common/protocol.go) //TODO

---

### `rate_limiter.go`

[`rate_limiter.go`](./common/rate_limiter.go) //TODO

---

### `topic.go`

[`topic.go`](./common/topic.go) //TODO

---

## Versioned

For details about the divergence between versions please consult the `README`s of each version package.

- [version 0](./v0)
- [version 1](./v1)

### Version `const.go`

`const.go` // TODO

---

### Version `init.go`

`init.go` // TODO

---

### Version `message.go`

`message.go` // TODO

---

### Version `peer.go`

`peer.go` // TODO

---

### Version `status_options.go`

`status_options.go` // TODO