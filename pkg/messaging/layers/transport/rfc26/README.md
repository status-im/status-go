# rfc26 — WakuMessage payload encryption

Implements the payload encryption codec specified by
**[26/WAKU2-PAYLOAD](https://lip.logos.co/messaging/draft/26/payload.html)**
(symmetric AES-256-GCM or asymmetric ECIES, plus an optional ECDSA signature,
over a padded frame). The spec covers the wire format; this README covers why
the layer matters in Status. The package always encrypts/decrypts and is
independent of the WakuMessage `version` field — see the package doc.

## Why this layer is not redundant

It's tempting to call this layer redundant: Status already encrypts message
content end-to-end with X3DH + Double Ratchet, and signs at the app layer. For
**confidentiality of the content**, that's true. But on the encrypted (1:1 and
private-group) paths this layer also conceals **metadata**, and that part is
*not* covered by anything above it.

The app layer only encrypts the innermost `EncryptedMessageProtocol.payload`.
The surrounding `ProtocolMessage` framing travels in cleartext within that
structure, and it carries linkable identifiers:

- `installation_id` — a **stable per-device sender identifier**
- `DR_header` — the Double Ratchet public key and message counters (`n`, `pn`),
  which link and order messages within a session
- `X3DH_header` — ephemeral key and bundle IDs

Today that whole `ProtocolMessage` protobuf is the *inner plaintext* of this
layer's encryption. For asymmetric (ECIES) sends a network observer sees only
`ephemeral_pubkey ‖ ciphertext ‖ HMAC`, and the fresh per-message ephemeral key
gives **recipient unlinkability** (recipients trial-decrypt). Remove this layer
and ship the payload as-is (`version=0`), and `installation_id`, the ratchet
header, etc. go on the wire in clear — a real metadata regression for private
chats.

So this layer is a genuine privacy boundary, not dead weight.

## The exception: public / community-broadcast chats

For public chats the symmetric key is derived deterministically from the
chat name (publicly computable), so this layer adds no confidentiality there —
and the payload is the cleartext `public_message` anyway. The per-message ECDSA
signature on these paths is, if anything, an identity *leak* rather than a
protection. The metadata argument above applies to the encrypted private paths,
not these.

## Removing it safely

Dropping this layer (the eventual `version=0` / logos-delivery cutover) must be
paired with moving the metadata protection up into the app layer first —
header-encrypting the Double Ratchet header and folding `installation_id` into
the encrypted payload, plus length-obfuscation padding — otherwise it's a
metadata regression for private chats. Tracked in
[logos-messaging/pm#420](https://github.com/logos-messaging/pm/issues/420);
background discussion in
[status-go#7493](https://github.com/status-im/status-go/issues/7493).

## Entry points

- `Encode(payload, symKey, pubKey, sigKey)` — send side; pass already-resolved
  keys (exactly one of symKey/pubKey; sigKey optional).
- `Decode(data, keyInfo)` — receive side; decrypts per `keyInfo.Kind`.

Relocated from go-waku's `waku/v2/payload`; output is byte-identical
(status-im/status-go#7462).
