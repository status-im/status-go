Receive messages using Waku Filter
===
WakuFilter is a protocol that enables subscribing to messages that a peer receives. You can find Waku Filter's specifications on [Vac RFC](https://rfc.vac.dev/spec/12/). This is a more lightweight version of WakuRelay specifically designed for bandwidth restricted devices. This is due to the fact that light nodes subscribe to full-nodes and only receive the messages they desire.

## Create a waku instance
```go
package main

import (
	"context"
    "fmt"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
)

...
wakuNode, err := node.New(node.WithWakuFilter(false))
if err != nil {
    fmt.Println(err)
    return
}

if err := wakuNode.Start(context.Background()); err != nil {
    fmt.Println(err)
    return
}
...

```

### Options
One of these options must be specified when instantiating a node supporting the waku relay protocol

- `WitWakuFilter(isFullNode, opts ...pubsub.Option)` - enables the waku filter protocol and receives an optional list of options to configure the protocol

## Adding a peer and receiving messages
```go
...

peerAddr, err := multiaddr.NewMultiaddr("/dns4/node-01.do-ams3.waku.test.status.im/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W")
if err != nil {
    panic(err)
}

_, err = wakuNode.AddPeer(peerAddr, string(filter.FilterID_v20beta1))
if err != nil {
    panic(err)
}

// Setting the content filter
cf := filter.ContentFilter{
    Topic:         "/waku/2/default-waku/proto",
    ContentTopics: []string{contentTopic},
}

_, subscription, err := wakuNode.Filter().Subscribe(context.Background(), cf)
if err != nil {
    panic(err)
}

for env := range subscription.Chan {
    fmt.Println("Received msg:", string(value.Message().Payload))
}
...
```
To receive messages sent via the relay protocol, you need to subscribe to a pubsub topic. This can be done via any of these functions:
- `wakuNode.Filter().Subscribe(ctx, contentFilter, opts)` - subscribes to receive messages that match a specific contentFilter

This function return a `Filter` subscrition. To stop receiving messages in this channel `wakuNode.UnsubscribeByFilter(ctx, theFilter)` can be executed which will close the subscription and channel for the filter. `wakuNode.Unsubscribe` and `wakuNode.UnsubscribeFilterByID` are also available for similar purposes

If no options are specified when subscribing to a filter node, go-waku will automatically choose the peer to subscribe to. This behaviour can be controlled via options:

### Options

- `filter.WithPeer(peerID)` - use an specific peer ID (which should be part of the node peerstore) to receive the messages from
- `filter.WithAutomaticPeerSelection(host)` - automatically select a peer that supports filter protocol from the peerstore to receive the messages from
- `filter.WithFastestPeerSelection(ctx)` - automatically select a peer based on its ping reply time
