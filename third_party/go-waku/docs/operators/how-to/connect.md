# Connect to other peers

*Note that this tutorial describes how to **configure** a node to connect to other peers before runnning the node.
For connecting a running node to existing peers,
see the [JSON-RPC API](https://rfc.vac.dev/spec/16/).*

There are currently three options.
Note that each of these options can be used in combination with any of the other two.
In other words, it is possible to configure a node to connect
to a static list of peers and
to discover such peer lists using DNS discovery and
discover and connect to random peers using discovery v5 with a bootstrap node.

## Option 1: Configure peers statically

Static peers can be provided to a go-waku node on startup using the `--staticnode` CLI parameter.
The `--staticnode` option can be repeated for each peer you want to connect to on startup.

```sh
./build/waku \
  --staticnode=<libp2p-multiaddr-peer1> \
  --staticnode=<libp2p-multiaddr-peer2>
```

As an example, consider a go-waku node that connects to two known peers
on the same local host (with IP `0.0.0.0`)
with TCP ports `60002` and `60003`,
and peer IDs `16Uiu2HAkzjwwgEAXfeGNMKFPSpc6vGBRqCdTLG5q3Gmk2v4pQw7H` and `16Uiu2HAmFBA7LGtwY5WVVikdmXVo3cKLqkmvVtuDu63fe8safeQJ` respectively.

```sh
./build/waku \
  --staticnode=/ip4/0.0.0.0/tcp/60002/p2p/16Uiu2HAkzjwwgEAXfeGNMKFPSpc6vGBRqCdTLG5q3Gmk2v4pQw7H \
  --staticnode=/ip4/0.0.0.0/tcp/60003/p2p/16Uiu2HAmFBA7LGtwY5WVVikdmXVo3cKLqkmvVtuDu63fe8safeQJ
```

## Option 2: Discover peers using DNS discovery

A node can discover other nodes to connect to using DNS-based discovery.
For a quickstart guide on how to configure DNS discovery,
see [this tutorial](./configure-dns-disc.md).
There is also a [more comprehensive tutorial](../../tutorial/dns-disc.md) for advanced users.

## Option 3: Discover peers using Waku Discovery v5

<!-- TODO: add link to a separate discv5 config tutorial here -->

Enable Discovery v5 using the `--discv5-discovery` option.

It is possible to configure bootstrap entries for the Discovery v5 routing table
using the `--discv5-bootstrap-node` option repeatedly.

```sh
./build/waku \
  --discv5-discovery=true \
  --discv5-bootstrap-node=<discv5-enr-bootstrap-entry1> \
  --discv5-bootstrap-node=<discv5-enr-bootstrap-entry2>
```

Note that if Discovery v5 is enabled and used in conjunction with DNS-based discovery,
the go-waku node will attempt to bootstrap the Discovery v5 routing table with ENRs extracted from the peers discovered via DNS.
