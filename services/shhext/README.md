Whisper API Extension
=====================

API
---

#### shhext_post

Accepts same input as shh_post (see https://github.com/ethereum/wiki/wiki/JSON-RPC#shh_post)

##### Returns

`DATA`, 32 Bytes - the envelope hash

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
