

branch | build status
-------|-------------
master | [![Build Status](https://travis-ci.org/status-im/status-go.svg?branch=master)](https://github.com/status-im/status-go/tree/master)
develop | [![Build Status](https://travis-ci.org/status-im/status-go.svg?branch=develop)](https://github.com/status-im/status-go/tree/develop)

# Status bindings for go-ethereum

- [How To Build](https://github.com/status-im/status-go/wiki/Build-Process-Explained)
- [Notes on Bindings](https://github.com/status-im/status-go/wiki/Notes-on-Bindings)

# LES protocol and referece server

- In order for clients to sync/pull blockchain headers using LES protocol, the full LES server is required, for reference.
- We expose one such server (to be added using `admin.addPeer()` or as `static-nodes.json`):
```json
[
  "enode://4e2bb6b09aa34375ae2df23fa063edfe7aaec952dba972449158ae0980a4abd375aca3c06a519d4f562ff298565afd288a0ed165944974b2557e6ff2c31424de@138.68.73.175:30303"
]
```
