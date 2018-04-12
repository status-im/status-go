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

Sends following event once per envelope.

```json
{
  "type": "envelope.sent",
  "event": {
    "hash": "0xea0b93079ed32588628f1cabbbb5ed9e4d50b7571064c2962c3853972db67790"
  }
}
```
