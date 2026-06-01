# rfc26 — WakuMessage version=1 payload codec

This package implements the WakuMessage `version=1` payload encoding used on
the Status wire, specified by **[26/WAKU2-PAYLOAD](https://rfc.vac.dev/spec/26/)**
("Waku Message Payload Encryption").

It is the legacy transport-boundary codec: it concatenates
`flags + payload-length + payload + padding + [signature]` and encrypts the
result — **AES-256-GCM** for symmetric keys, **ECIES** for asymmetric keys —
with an optional **ECDSA** signature over the payload. Despite the package
sitting under `transport/`, this layer performs real encryption and signing,
not mere serialization.

The format derives from [EIP-627 (Whisper)](https://eips.ethereum.org/EIPS/eip-627)
and the [RLPx ECIES spec](https://github.com/ethereum/devp2p/blob/master/rlpx.md#ecies-encryption).
It was relocated into status-go from go-waku's `waku/v2/payload` package
(status-im/status-go#7462); byte output is identical, so existing network
traffic stays interoperable.

## Status

This wire-format layer is cryptographically redundant with the X3DH /
Double Ratchet (`layers/encryption`) and app-layer signatures that already
protect Status messages. It is retained only for wire-compatibility with the
existing network. Dropping it (and the `version=0` migration that enables it)
is tracked in [logos-messaging/pm#420](https://github.com/logos-messaging/pm/issues/420).

## Entry points

- `EncodeV1(payload, symKey, pubKey, sigKey)` — send side; pass already-resolved keys.
- `Decode(version, data, keyInfo)` — receive side; `version=0` returns the payload unchanged.
- `Payload{...}.Encode(version)` — lower-level encode used by `EncodeV1`.
