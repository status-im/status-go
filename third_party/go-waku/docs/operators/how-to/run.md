# Running go-waku

```sh
# Run with default configuration
./build/waku

# See available command line options
./build/waku --help
```

## Default configuration

By default a go-waku node will:
- generate a new private key and libp2p identities after every restart.
See [this tutorial](./configure-key.md) if you want to generate and configure a persistent private key.
- listen for incoming libp2p connections on the default TCP port (`60000`)
- enable `relay` protocol
- subscribe to the default pubsub topic, namely `/waku/2/default-waku/proto`
- enable `store` protocol, but only as a client.
This implies that the go-waku node will not persist any historical messages itself,
but can query `store` service peers who do so.
To configure `store` as a service node,
see [this tutorial](./configure-store.md).

> **Note:** The `filter` and `lightpush` protocols are _not_ enabled by default.
Consult the [configuration guide](./configure.md) on how to configure your go-waku node to run these protocols.

Some typical non-default configurations are explained below.
For more advanced configuration, see the [configuration guide](./configure.md).
Different ways to connect to other nodes are expanded upon in our [connection guide](./connect.md).

## Finding your listening address(es)

Find the log entry beginning with `Listening on`.
It should be printed at INFO level when you start your node
and contains a list of all publically announced listening addresses for the go-waku node.

For example

```
2022-07-25T07:26:01.150-0400    INFO    gowaku.node2    listening       {"node": "16Uiu2HAkxFXYTRHWTnT1oT2BA8foKopFSXNfvXzbfvcuF6e88qf4", "multiaddr": ["/ip4/127.0.0.1/tcp/60000/p2p/16Uiu2HAkxFXYTRHWTnT1oT2BA8foKopFSXNfvXzbfvcuF6e88qf4", "/ip4/127.0.0.1/tcp/60001/ws/p2p/16Uiu2HAkxFXYTRHWTnT1oT2BA8foKopFSXNfvXzbfvcuF6e88qf4"]}
```

indicates that your node is listening on the TCP transport addresses

```
/ip4/127.0.0.1/tcp/60000/p2p/16Uiu2HAkxFXYTRHWTnT1oT2BA8foKopFSXNfvXzbfvcuF6e88qf4
```

and websocket address

```
/ip4/127.0.0.1/tcp/60001/ws/p2p/16Uiu2HAkxFXYTRHWTnT1oT2BA8foKopFSXNfvXzbfvcuF6e88qf4
```

You can also query a running node for its listening addresses
using a [`get_waku_v2_debug_v1_info` JSON-RPC API](https://rfc.vac.dev/spec/16/#get_waku_v2_debug_v1_info) call.

For example

```sh
curl -d '{"jsonrpc":"2.0","id":"id","method":"get_waku_v2_debug_v1_info", "params":[]}' --header "Content-Type: application/json" http://localhost:8545
```

returns a response similar to

```json
{
  "result": {
    "enrUri": "enr:-Iu4QJecqtDmg5JBwhEGCifJE-nfBUPvJpV1_Q7CtbJqX85pc8TV5xNIJKohJHnOtbQjycQV0OSzJeCsUB2a7hnfEP0BgmlkgnY0gmlwhMCoAG2Jc2VjcDI1NmsxoQJyDYLm_cOh10d-9TP34svDeh_AsrfmoDqrlpDeoNOmg4N0Y3CC6mCFd2FrdTIB",
    "listenAddresses": [
      "/ip4/192.168.0.109/tcp/60000/p2p/16Uiu2HAm36tQZYF9ijH9PzgZKcJDxyKz93iue4RjpBLkvcbo6mhU",
      "/ip4/127.0.0.1/tcp/60000/p2p/16Uiu2HAm36tQZYF9ijH9PzgZKcJDxyKz93iue4RjpBLkvcbo6mhU"
    ]
  },
  "error": null,
  "id": "id"
}
```

## Finding your discoverable ENR address(es)

A go-waku node can encode its addressing information in an [Ethereum Node Record (ENR)](https://eips.ethereum.org/EIPS/eip-778) according to [`31/WAKU2-ENR`](https://rfc.vac.dev/spec/31/).
These ENR are most often used for discovery purposes.

### ENR for DNS discovery and DiscV5

Find the log entry containing the text `enr record`.

For example

```
2022-07-25T07:27:15.007-0400    INFO    gowaku.node2    enr record      {"node": "16Uiu2HAmSBY66Awj56ssci4tJ3bEmcG8oRpufZwqe4Ueb46i7bWg", "enr": "enr:-KO4QJmGMGthIh_kCubluVD9jZLPDcqNgLgDYJxruIULs2elNcZxnIYqEZD-f9f-zsY2QMqEVosMxShxwTG8BkzkWQ8BgmlkgnY0gmlwhMCoAG2KbXVsdGlhZGRyc4wACgTAqABtBuph3QOJc2VjcDI1NmsxoQPI-z2SDgsKlci7pAYysALdIFv9ySJlRpynQbZdlJoU4YN0Y3CC6mCFd2FrdTIB"}
```

indicates that your node addresses are encoded in the ENR

```
enr:-KO4QJmGMGthIh_kCubluVD9jZLPDcqNgLgDYJxruIULs2elNcZxnIYqEZD-f9f-zsY2QMqEVosMxShxwTG8BkzkWQ8BgmlkgnY0gmlwhMCoAG2KbXVsdGlhZGRyc4wACgTAqABtBuph3QOJc2VjcDI1NmsxoQPI-z2SDgsKlci7pAYysALdIFv9ySJlRpynQbZdlJoU4YN0Y3CC6mCFd2FrdTIB
```

## Typical configuration (relay node)

The typical configuration for a go-waku node is to run the `relay` protocol,
subscribed to the default pubsub topic `/waku/2/default-waku/proto`,
and connecting to one or more existing peers.
We assume below that running nodes also participate in Discovery v5
to continually discover and connect to random peers for a more robust mesh.

### Connecting to known peer(s)

A typical run configuration for a go-waku node is to connect to existing peers with known listening addresses using the `--staticnode` option.
The `--staticnode` option can be repeated for each peer you want to connect to on startup.
This is also useful if you want to run several go-waku instances locally
and therefore know the listening addresses of all peers.

As an example, consider a go-waku node that connects to two known peers
on the same local host (with IP `0.0.0.0`)
with TCP ports `60002` and `60003`,
and peer IDs `16Uiu2HAkzjwwgEAXfeGNMKFPSpc6vGBRqCdTLG5q3Gmk2v4pQw7H` and `16Uiu2HAmFBA7LGtwY5WVVikdmXVo3cKLqkmvVtuDu63fe8safeQJ` respectively.
The Discovery v5 routing table can similarly be bootstrapped using a static ENR.
We include an example below.

```sh
./build/waku \
  --staticnode=/ip4/0.0.0.0/tcp/60002/p2p/16Uiu2HAkzjwwgEAXfeGNMKFPSpc6vGBRqCdTLG5q3Gmk2v4pQw7H \
  --staticnode=/ip4/0.0.0.0/tcp/60003/p2p/16Uiu2HAmFBA7LGtwY5WVVikdmXVo3cKLqkmvVtuDu63fe8safeQJ \
  --discv5-discovery=true \
  --discv5-bootstrap-node=enr:-JK4QM2ylZVUhVPqXrqhWWi38V46bF2XZXPSHh_D7f2PmUHbIw-4DidCBnBnm-IbxtjXOFbdMMgpHUv4dYVH6TgnkucBgmlkgnY0gmowhCJ6_HaJc2VjcDI1NmsxoQM06FsT6EJ57mzR_wiLu2Bz1dER2nUFSCpaFzCccQtnhYN0Y3CCdl-DdWRwgiMohXdha3UyDw
```


### Connecting to the `waku.sandbox` network

You can use DNS discovery to bootstrap connection to the existing production network.
Discovery v5 will attempt to extract the ENRs of the discovered nodes as bootstrap entries to the routing table.

```sh
./build/waku \
  --dns-discovery=true \
  --dns-discovery-url=enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im \
  --discv5-discovery=true
```

### Connecting to the `waku.test` network

You can use DNS discovery to bootstrap connection to the existing test network.
Discovery v5 will attempt to extract the ENRs of the discovered nodes as bootstrap entries to the routing table.

```sh
./build/waku \
  --dns-discovery=true \
  --dns-discovery-url=enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im \
  --discv5-discovery=true
```

## Typical configuration (relay and store service node)

Often go-waku nodes choose to also store historical messages
from where it can be queried by other peers who may have been temporarily offline.
For example, a typical configuration for such a store service node,
[connecting to the `waku.test`](#connecting-to-the-wakutest-network) fleet on startup,
appears below.

```sh
./build/waku \
  --store=true \
  --persist-messages=true \
  --db-path=/mnt/go-waku/data/db1/ \
  --store-capacity=150000 \
  --dns-discovery=true \
  --dns-discovery-url=enrtree://AOGYWMBYOUIMOENHXCHILPKY3ZRFEULMFI4DOM442QSZ73TT2A7VI@test.waku.nodes.status.im \
  --discv5-discovery=true
```

See our [store configuration tutorial](./configure-store.md) for more.

## Interact with a running go-waku node

A running go-waku node can be interacted with using the [Waku v2 JSON RPC API](https://rfc.vac.dev/spec/16/).

> **Note:** Private and Admin API functionality are disabled by default.
To configure a go-waku node with these enabled,
use the `--rpc-admin:true` and `--rpc-private:true` CLI options.
