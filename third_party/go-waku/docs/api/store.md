Retrieve message history using Waku Store
===

DApps running on a phone or in a browser are often offline: The browser could be closed or mobile app in the background.

[Waku Relay](https://rfc.vac.dev/spec/11/) is a gossip protocol. As a user, it means that your peers forward you messages they just received. If you cannot be reached by your peers, then messages are not relayed; relay peers do not save messages for later.

However, [Waku Store](https://rfc.vac.dev/spec/13/) peers do save messages they relay, allowing you to retrieve them at a later time. The Waku Store protocol is best-effort and does not guarantee data availability. Waku Relay should still be preferred when online; Waku Store can be used after resuming connectivity: For example, when the dApp starts.


```go
import (
    "context"
    
    "github.com/go-waku/go-waku/waku/v2/protocol/store"
)

...
query := store.Query{
    Topic: ..., // optional pubsub topic
    ContentTopics: []string{...}, // optional content topics
    StartTime: ..., // optional start time in ms
    EndTime: ..., // optional end time in ms
}

result, err := wakuNode.Store().Query(context.Background(), query, WithPaging(true, 20));
if err != nil {
    // Handle error ...
}

for !result.IsComplete() {
    for _, msg := range result.GetMessages() {
        // Do something with the messages
    }

    err := result.Next(ctx)
    if err != nil {
        // Handle error ...
    }
}
```

To retrieve message history, a `store.Query` struct should be created with the attributes to filter the messages. This struct should be passed to `wakuNode.Store().Query`. A successful execution will return a `store.Result` that can be used to retrieve more messages if pagination is being used. `wakuNode.Store().Next` should be used if the number of messages in the `store.Result` is greater than 0.

The query function also accepts a list of options:

### Options
- `store.WithPeer(peerID)` - use an specific peer ID (which should be part of the node peerstore) to broadcast the message with 
- `store.WithAutomaticPeerSelection(host)` - automatically select a peer that supports store protocol from the peerstore to broadcast the message with
- `store.WithFastestPeerSelection(ctx)` - automatically select a peer based on its ping reply time
- `store.WithCursor(index)` - use cursor to retrieve messages starting from the index of a WakuMessage. This cursor can be obtained from a `store.Result` obtained from `wakuNode.Store().Query` 
- `store.WithPaging(asc, pageSize)` - specify the order and maximum number of records to return

## Filter messages

### By timestamp
By default, Waku Store nodes keep messages for 30 days. Depending on your use case, you may not need to retrieve 30 days worth of messages. Waku Message defines an optional unencrypted timestamp field. The timestamp is set by the sender. 
You can filter messages that include a timestamp within given bounds with the `StartTime` and `EndTime` attributes of `store.Query`. If a message does have their timestamp attribute defined, the time when they were stored in the node will be used instead.

```go
// 7 days/week, 24 hours/day, 60min/hour, 60secs/min, 100ms/sec
query := store.Query{
    StartTime:  utils.GetUnixEpoch() - 7 * 24 * 60 * 60 * 1000,
    EndTime:  utils.GetUnixEpoch()
}

result, err := wakuNode.Store().Query(context.Background(), query);
```

### By topic
It is possible to filter the messages by the pubsub topic in which they were published, and by their content topic. The `ContentTopics` attribute maps to the `contentTopic` field of a `WakuMessage`. Multiple content topics can be specified. Leaving these fields empty will retrieve messages regardless on the pubsub topic they were published and content topic they contain.
```go
query := store.Query{
    Topic: "some/pubsub/topic",
    ContentTopics: []string{"content/topic/1", "content/topic/2"},
}

result, err := wakuNode.Store().Query(context.Background(), query);
```