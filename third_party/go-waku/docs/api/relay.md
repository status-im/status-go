Receive and send messages using Waku Relay
===

Waku Relay is a gossip protocol that enables you to send and receive messages. You can find Waku Relay’s specifications on [Vac RFC](https://rfc.vac.dev/spec/11/).

## Create a waku instance
```go
package main

import (
	"context"
    "fmt"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

...
wakuNode, err := node.New(node.WithWakuRelay())
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

- `WithWakuRelay(opts ...pubsub.Option)` - enables the waku relay protocol and receives an optional list of pubsub options to tune or configure the gossipsub parameters. Supported options can be seen [here](https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub#Option). The recommended [parameter configuration](https://rfc.vac.dev/spec/29/) is used by default.
- `WithWakuRelayAndMinPeers(minRelayPeersToPublish int, opts ...pubsub.Option)` - enables the waku relay protocol, specifying the minimum number of peers a topic should have to send a message. It also receives an optional list of pubsub [options](https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub#Option)



## Receiving messages
```go
...
contentFilter := protocol.NewContentFilter(relay.DefaultWakuTopic)
sub, err := wakuNode.Relay().Subscribe(context.Background, contentFilter) ([]*Subscription, error)
if err != nil {
    fmt.Println(err)
    return
}

for value := range sub[0].C {
    fmt.Println("Received msg:", string(value.Message().Payload))
}
...
```
To receive messages sent via the relay protocol, you need to subscribe specifying a content filter with the function `Subscribe(ctx context.Context, contentFilter waku_proto.ContentFilter, opts ...RelaySubscribeOption) ([]*Subscription, error)`. This functions return a list of `Subscription` struct containing a channel on which messages will be received. To stop receiving messages `WakuRelay`'s `Unsubscribe(ctx context.Context, contentFilter waku_proto.ContentFilter) error` can be executed which will close the channel (without unsubscribing from the pubsub topic) which will make sure the subscription is stopped, and if no other subscriptions exist for underlying pubsub topic, the pubsub is also unsubscribed.

## Sending messages

```go
import (
    "github.com/waku-org/go-waku/waku/v2/protocol/pb"
    "github.com/waku-org/go-waku/waku/v2/utils"
)

...
msg := &pb.WakuMessage{
    Payload:      []byte("Hello World"),
    Version:      0,
    ContentTopic: protocol.NewContentTopic("basic2", 1, "test", "proto").String(),
    Timestamp:    utils.GetUnixEpoch(),
}

msgId, err = wakuNode.Relay().Publish(context.Background(), msg)
if err != nil {
    log.Error("Error sending a message: ", err)
}
```

To send a message, it needs to be wrapped into a [`WakuMessage`](https://rfc.vac.dev/spec/14/) protobuffer. The payload of the message is not limited to strings. Any kind of data that can be serialized
into a `[]byte` can be sent as long as it does not exceed the maximum length a message can have (~1MB)

`wakuNode.Relay().Publish(ctx, msg, opts...)` is used to publish a message. This function will return a message id on success, or an error if the message could not be published.

If no options are specified, go-waku will automatically choose the peer used to broadcast the message via Relay and publish the message to a pubsub topic derived from the content topic of the message. This behaviour can be controlled via options:

### Options
- `relay.WithPubSubTopic(topic)` - broadcast the message using a custom pubsub topic
- `relay.WithDefaultPubsubTopic()` - broadcast the message to the default pubsub topic

> If `WithWakuRelayAndMinPeers` was used during the instantiation of the wakuNode, it should be possible to verify if there's enough peers for publishing to a topic with `wakuNode.Relay().EnoughPeersToPublish()` and `wakuNode.Relay().EnoughPeersToPublishToTopic(topic)`


## Stop receiving messages
```go
...
err := wakuNode.Relay().Unsubscribe(context.Background(), theTopic)
if err != nil {
    fmt.Println(err)
    return
}
...
```
> To stop receiving messages from the default topic, use `relay.DefaultWakuTopic` as the topic parameter

