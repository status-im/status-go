# Configure store protocol

Store protocol is enabled by default on a go-waku node.
This is controlled by the `--store` CLI option.

```sh
# Disable store protocol on startup
./build/waku --store=false
```

Note that this only mounts the `store` protocol,
meaning your node will indicate to other peers that it supports `store`.
It does not yet allow your node to either retrieve historical messages as a client
or store and serve historical messages itself.

## Configuring a store client

Ensure that `store` is enabled (this is `true` by default) and provide at least one store service node address with the `--storenode` CLI option.

See the following example, using the peer at `/dns4/node-01.ac-cn-hongkong-c.waku.test.status.im/tcp/30303/p2p/16Uiu2HAkzHaTP5JsUwfR9NR8Rj9HC24puS6ocaU8wze4QrXr9iXp` as store service node.

```sh
./build/waku \
  --store=true \
  --storenode=/dns4/node-01.ac-cn-hongkong-c.waku.test.status.im/tcp/30303/p2p/16Uiu2HAkzHaTP5JsUwfR9NR8Rj9HC24puS6ocaU8wze4QrXr9iXp
```

Your node can now send queries to retrieve historical messages
from the configured store service node.
One way to trigger such queries is asking your node for historical messages using the [Waku v2 JSON RPC API](https://rfc.vac.dev/spec/16/).

## Configuring a store service node

To store historical messages on your node which can be served to store clients the `--persist-messages` CLI option must be enabled.
By default a node would store up to the latest `50 000` messages received in the last `2 592 000` seconds (30 days)
This is configurable using the `--store-capacity` and `store-seconds` options.
A node that has a `--db-path` set will backup historical messages to a local database at the DB path
and persist these messages even after a restart.

```sh
./build/waku \
  --store=true \
  --persist-messages=true \
  --db-path=/mnt/go-waku/data/db1/ \
  --store-capacity=150000
  --store-seconds=5000000
```
