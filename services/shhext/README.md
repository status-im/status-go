Whisper API Extension
=====================

API
---

#### shhext_post

Accepts same input as shh_post (see https://github.com/ethereum/wiki/wiki/JSON-RPC#shh_post)

##### Returns

`DATA`, 32 Bytes - the envelope hash

#### shhext_requestMessages

Sends a request for historic messages to a mail server.

##### Parameters

1. `Object` - The message request object:

- `mailServerPeer`:`URL` - Mail servers' enode addess
- `from`:`QUANTITY` - (optional) Lower bound of time range as unix timestamp, default is 24 hours back from now
- `to`:`QUANTITY`- (optional) Upper bound of time range as unix timestamp, default is now
- `topic`:`DATA`, 4 Bytes - Regular whisper topic
- `symKeyID`:`DATA`- ID of a symmetric key to authenticate to mail server, derived form mail server password

##### Returns

`Boolean` - returns `true` if the request was send, otherwise `false`.

Signals
-------

Sends sent signal once per envelope.

```json
{
  "type": "envelope.sent",
  "event": {
    "hash": "0xea0b93079ed32588628f1cabbbb5ed9e4d50b7571064c2962c3853972db67790"
  }
}
```

Sends expired signal if envelope dropped from whisper local queue before it was
sent to any peer on the network.

```json
{
  "type": "envelope.expired",
  "event": {
    "hash": "0x754f4c12dccb14886f791abfeb77ffb86330d03d5a4ba6f37a8c21281988b69e"
  }
}
```
