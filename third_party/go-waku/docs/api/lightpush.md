Send messages using Waku Lightpush
===

Waku Light Push enables a client to receive a confirmation when sending a message.

The Waku Relay protocol sends messages to connected peers but does not provide any information on whether said peers have received messages. This can be an issue when facing potential connectivity issues. For example, when the connection drops easily, or it is connected to a small number of relay peers.

Waku Light Push allows a client to get a response from a remote peer when sending a message. Note this only guarantees that the remote peer has received the message, it cannot guarantee propagation to the network.

It also means weaker privacy properties as the remote peer knows the client is the originator of the message. Whereas with Waku Relay, a remote peer would not know whether the client created or forwarded the message.

You can find Waku Light Push’s specifications on [Vac RFC](https://rfc.vac.dev/spec/19/).


## Create a waku instance
```go
package main

import (
	"context"
    "fmt"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
)

...
wakuNode, err := node.New(node.WithLightPush())
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
> If the node also has the relay protocol enabled, it will be able to receive lightpush request to broadcast messages to other relay nodes



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

msgId, err = wakuNode.Lightpush().Publish(context.Background(), msg)
if err != nil {
    log.Error("Error sending a message: ", err)
}
```


To send a message, it needs to be wrapped into a [`WakuMessage`](https://rfc.vac.dev/spec/14/) protobuffer.
The payload of the message is not limited to strings. Any kind of data that can be serialized
into a `[]byte` can be sent as long as it does not exceed the maximum length a message can have (~1MB)

`wakuNode.Lightpush().Publish(ctx, msg, opts...)` is used to publish a message. This function will return a message id on success, or an error if the message could not be published.

If no options are specified, go-waku will automatically choose the peer used to broadcast the message via Lightpush and publish the message to a pubsub topic derived from the content topic of the message. This behaviour can be controlled via options:

### Options
- `lightpush.WithPubSubTopic(topic)` - broadcast the message using a custom pubsub topic
- `lightpush.WithDefaultPubsubTopic()` - broadcast the message to the default pubsub topic
- `lightpush.WithPeer(peerID)` - use an specific peer ID (which should be part of the node peerstore) to broadcast the message with 
- `lightpush.WithAutomaticPeerSelection(host)` - automatically select a peer that supports lightpush protocol from the peerstore to broadcast the message with
- `lightpush.WithFastestPeerSelection(ctx)` - automatically select a peer based on its ping reply time
