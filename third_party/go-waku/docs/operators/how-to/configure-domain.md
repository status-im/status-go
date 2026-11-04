# Configure a domain name

> **Note:** This page describes configuring a domain name that resolves to your node's IP
and is unrelated to [DNS discovery](./configure-dns-disc.md),
by which a node may discover the listening addresses of other peers using DNS.

It is possible to configure an IPv4 DNS domain name that resolves to the node's public IPv4 address.

```shell
waku --dns4-domain-name=mynode.example.com
```

This allows for the node's publically announced `multiaddrs` to use the `/dns4` scheme.
In addition, nodes with domain name and [secure websocket configured](./configure-websocket.md),
will generate a discoverable ENR containing the `/wss` multiaddr with `/dns4` domain name.
This is necessary to verify domain certificates when connecting to this node over secure websocket.