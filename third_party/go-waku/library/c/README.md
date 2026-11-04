This specification describes the API for consuming go-waku when built as a dynamic or static library


# libgowaku.h


# Introduction

Native applications that wish to integrate Waku may not be able to use go-waku/nwaku and its JSON RPC API due to constraints
on packaging, performance or executables.

An alternative is to link existing Waku implementation as a static or dynamic library in their application.

This specification describes the C API that SHOULD be implemented by native Waku library and that SHOULD be used to
consume them.

# The API

## General

### `WakuMessageCallback` type

All the API functions require passing callbacks which will be executed depending on the result of the execution result.
These callbacks are defined as
```c
typedef void (*WakuCallBack) (int ret_code, const char* msg, void * user_data);
```
With ret_code containing a status code indicating the execution result, `msg` containing a `\0` terminated string. The format of the data sent to these callbacks will depend on the function being executed. The data for succesful results can be characters, numeric or json.
If the function being invoked returns an error, `msg` will contain the error string.

### Status Codes
The API functions return an integer with status codes depending on the execution result. The following status codes are defined:
- `0` - Success
- `1` - Error
- `2` - Missing callback

### `JsonMessage` type

A Waku Message in JSON Format:

```ts
{
    payload: string;
    contentTopic: string;
    version: number;
    timestamp: number;
}
```

Fields:

- `payload`: base64 encoded payload, [`waku_utils_base64_encode`](#extern-char-waku_utils_base64_encodechar-data) can be used for this.
- `contentTopic`: The content topic to be set on the message.
- `version`: The Waku Message version number.
- `timestamp`: Unix timestamp in nanoseconds.

### `DecodedPayload` type

A payload once decoded, used when a received Waku Message is encrypted:

```ts
interface DecodedPayload {
    pubkey?: string;
    signature?: string;
    data: string;
    padding: string;
  }
```

Fields:

- `pubkey`: Public key that signed the message (optional), hex encoded with `0x` prefix,
- `signature`: Message signature (optional), hex encoded with `0x` prefix,
- `data`: Decrypted message payload base64 encoded,
- `padding`: Padding base64 encoded.

### `ContentFilter` type

The criteria to create subscription in relay protocol or to a filter full node in JSON Format:

```ts
{
    contentTopics: string[];
    pubsubTopic?: string;
}
```

Fields:

- `contentTopics`: Array of content topics.
- `pubsubTopic`: Optional pubsub topic when using contentTopics as per Autosharding. In case of named or static-sharding, pubSub topic is mandatory.


### `LegacyFilterSubscription` type

The criteria to create subscription to a filter full node in JSON Format:

```ts
{
    contentFilters: LegacyContentFilter[];
    pubsubTopic: string?;
}
```

Fields:

- `contentFilters`: Array of [`LegacyContentFilter`](#legacycontentfilter-type) being subscribed to / unsubscribed from.
- `topic`: Optional pubsub topic.


### `LegacyContentFilter` type

```ts
{
    contentTopic: string;
}
```

Fields:

- `contentTopic`: The content topic of a Waku message.

### `StoreQuery` type

Criteria used to retrieve historical messages

```ts
interface StoreQuery {
    pubsubTopic?: string;
    contentTopics: string[];
    startTime?: number;
    endTime?: number;
    pagingOptions?: PagingOptions
  }
```

Fields:

- `pubsubTopic`: The pubsub topic on which messages are published.
- `contentTOpics`: Array of content topics to query for historical messages,
- `startTime`: The inclusive lower bound on the timestamp of queried messages. This field holds the Unix epoch time in nanoseconds.
- `endTime`: The inclusive upper bound on the timestamp of queried messages. This field holds the Unix epoch time in nanoseconds.
- `pagingOptions`: Paging information in [`PagingOptions`](#pagingoptions-type) format.

### `StoreResponse` type

The response received after doing a query to a store node:

```ts
interface StoreResponse {
    messages: JsonMessage[];
    pagingOptions?: PagingOptions;
  }
```
Fields:

- `messages`: Array of retrieved historical messages in [`JsonMessage`](#jsonmessage-type) format.
- `pagingOption`: Paging information in [`PagingOptions`](#pagingoptions-type) format from which to resume further historical queries

### `PagingOptions` type

```ts
interface PagingOptions {
    pageSize: number;
    cursor?: Index;
    forward: bool;
  }
```
Fields:

- `pageSize`: Number of messages to retrieve per page.
- `cursor`: Message Index from which to perform pagination. If not included and forward is set to true, paging will be performed from the beginning of the list. If not included and forward is set to false, paging will be performed from the end of the list.
- `forward`:  `true` if paging forward, `false` if paging backward

### `Index` type

```ts
interface Index {
    digest: string;
    receiverTime: number;
    senderTime: number;
    pubsubTopic: string;
  }
```

Fields:

- `digest`: Hash of the message at this [`Index`](#index-type).
- `receiverTime`: UNIX timestamp in nanoseconds at which the message at this [`Index`](#index-type) was received.
- `senderTime`: UNIX timestamp in nanoseconds at which the message is generated by its sender.
- `pubsubTopic`: The pubsub topic of the message at this [`Index`](#index-type).

## Events

Asynchronous events require a callback to be registered.
An example of an asynchronous event that might be emitted is receiving a message.
When an event is emitted, this callback will be triggered receiving a JSON string of type `JsonSignal`.

### `JsonSignal` type

```ts
{
    type: string;
    event: any;
}
```

Fields:

- `type`: Type of signal being emitted. Currently, only `message` is available.
- `event`: Format depends on the type of signal.

For example:

```json
{
  "type": "message",
  "event": {
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageId": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```

| `type`    | `event` Type       |
|:----------|--------------------|
| `message` | `JsonMessageEvent` | 

### `JsonMessageEvent` type

Type of `event` field for a `message` event:

```ts
{
    pubsubTopic: string;
    messageId: string;
    wakuMessage: JsonMessage;
}
```

- `pubsubTopic`: The pubsub topic on which the message was received.
- `messageId`: The message id.
- `wakuMessage`: The message in [`JsonMessage`](#jsonmessage-type) format.

### `extern void waku_set_event_callback(WakuCallBack cb)`

Register callback to act as event handler and receive application signals,
which are used to react to asynchronous events in Waku.

**Parameters**

1. `WakuCallBack cb`: callback that will be executed when an async event is emitted.
  The function signature for the callback should be `void myCallback(char* signal, void * user_data)`

## Node management

### `JsonConfig` type

Type holding a node configuration:

```ts
interface JsonConfig {
    host?: string;
    port?: number;
    advertiseAddr?: string;
    nodeKey?: string;
    keepAliveInterval?: number;
    relay?: boolean;
    relayTopics?: Array<string>;
    gossipsubParameters?: GossipSubParameters;
    minPeersToPublish?: number;
    legacyFilter?: boolean;
    discV5?: boolean;
    discV5BootstrapNodes?: Array<string>;
    discV5UDPPort?: number;
    store?: boolean;
    databaseURL?: string;
    storeRetentionMaxMessages?: number;
    storeRetentionTimeSeconds?: number;
    websocket?: Websocket;
    dns4DomainName?: string;
    clusterID: int;
    shards: Array<uint16>;
}
```

Fields: 

All fields are optional.
If a key is `undefined`, or `null`, a default value will be set.

- `host`: Listening IP address.
  Default `0.0.0.0`.
- `port`: Libp2p TCP listening port.
  Default `60000`. 
  Use `0` for random.
- `advertiseAddr`: External address to advertise to other nodes.
  Can be ip4, ip6 or dns4, dns6.
  If `null`, the multiaddress(es) generated from the ip and port specified in the config (or default ones) will be used.
  Default: `null`.
- `nodeKey`: Secp256k1 private key in Hex format (`0x123...abc`).
  Default random.
- `keepAliveInterval`: Interval in seconds for pinging peers to keep the connection alive.
  Default `20`.
- `relay`: Enable relay protocol.
  Default `true`.
- `relayTopics`:  Array of pubsub topics that WakuRelay will automatically subscribe to when the node starts
  Default `[]`
- `gossipSubParameters`: custom gossipsub parameters. See `GossipSubParameters` section for defaults
- `minPeersToPublish`: The minimum number of peers required on a topic to allow broadcasting a message.
  Default `0`.
- `legacyFilter`: Enable Legacy Filter protocol.
  Default `false`.
- `discV5`: Enable DiscoveryV5.
  Default `false`
- `discV5BootstrapNodes`: Array of bootstrap nodes ENR
- `discV5UDPPort`: UDP port for DiscoveryV5
  Default `9000`
- `store`: Enable store protocol to persist message history
  Default `false`
- `databaseURL`: url connection string. Accepts SQLite and PostgreSQL connection strings
  Default: `sqlite3://store.db`
- `storeRetentionMaxMessages`: max number of messages to store in the database.
  Default `10000`
- `storeRetentionTimeSeconds`: max number of seconds that a message will be persisted in the database.
  Default `2592000` (30d)
- `websocket`: custom websocket support parameters. See `Websocket` section for defaults
- `dns4DomainName`: the domain name resolving to the node's public IPv4 address.


For example:
```json
{
  "host": "0.0.0.0",
  "port": 60000,
  "advertiseAddr": "1.2.3.4",
  "nodeKey": "0x123...567",
  "keepAliveInterval": 20,
  "relay": true,
  "minPeersToPublish": 0
}
```


### `GossipsubParameters` type

Type holding custom gossipsub configuration:

```ts
interface GossipSubParameters {
    D?: number;
    D_low?: number;
    D_high?: number;
    D_score?: number;
    D_out?: number;
    HistoryLength?: number;
    HistoryGossip?: number;
    D_lazy?: number;
    GossipFactor?: number;
    GossipRetransmission?: number;
    HeartbeatInitialDelayMs?: number;
    HeartbeatIntervalSeconds?: number;
    SlowHeartbeatWarning?: number;
    FanoutTTLSeconds?: number;
    PrunePeers?: number;
    PruneBackoffSeconds?: number;
    UnsubscribeBackoffSeconds?: number;
    Connectors?: number;
    MaxPendingConnections?: number;
    ConnectionTimeoutSeconds?: number;
    DirectConnectTicks?: number;
    DirectConnectInitialDelaySeconds?: number;
    OpportunisticGraftTicks?: number;
    OpportunisticGraftPeers?: number;
    GraftFloodThresholdSeconds?: number;
    MaxIHaveLength?: number;
    MaxIHaveMessages?: number;
    IWantFollowupTimeSeconds?: number;
}
```

Fields: 

All fields are optional.
If a key is `undefined`, or `null`, a default value will be set.

- `d`: optimal degree for a GossipSub topic mesh. 
  Default `6`
- `dLow`: lower bound on the number of peers we keep in a GossipSub topic mesh
  Default `5`
- `dHigh`: upper bound on the number of peers we keep in a GossipSub topic mesh.
  Default `12`
- `dScore`: affects how peers are selected when pruning a mesh due to over subscription.
  Default `4`
- `dOut`: sets the quota for the number of outbound connections to maintain in a topic mesh.
  Default `2`
- `historyLength`: controls the size of the message cache used for gossip.
  Default `5`
- `historyGossip`: controls how many cached message ids we will advertise in IHAVE gossip messages.
  Default `3`
- `dLazy`: affects how many peers we will emit gossip to at each heartbeat.
  Default `6`
- `gossipFactor`: affects how many peers we will emit gossip to at each heartbeat.
  Default `0.25`
- `gossipRetransmission`: controls how many times we will allow a peer to request the same message id through IWANT gossip before we start ignoring them.
  Default `3`
- `heartbeatInitialDelayMs`: short delay in milliseconds before the heartbeat timer begins after the router is initialized.
  Default `100` milliseconds
- `heartbeatIntervalSeconds`: controls the time between heartbeats.
  Default `1` second
- `slowHeartbeatWarning`: duration threshold for heartbeat processing before emitting a warning.
  Default `0.1`
- `fanoutTTLSeconds`: controls how long we keep track of the fanout state.
  Default `60` seconds
- `prunePeers`: controls the number of peers to include in prune Peer eXchange.
  Default `16`
- `pruneBackoffSeconds`: controls the backoff time for pruned peers.
  Default `60` seconds
- `unsubscribeBackoffSeconds`: controls the backoff time to use when unsuscribing from a topic.
  Default `10` seconds
- `connectors`: number of active connection attempts for peers obtained through PX.
  Default `8`
- `maxPendingConnections`: maximum number of pending connections for peers attempted through px. 
  Default `128`
- `connectionTimeoutSeconds`: timeout in seconds for connection attempts.
  Default `30` seconds
- `directConnectTicks`: the number of heartbeat ticks for attempting to reconnect direct peers that are not currently connected.
  Default `300`
- `directConnectInitialDelaySeconds`: initial delay before opening connections to direct peers.
  Default `1` second
- `opportunisticGraftTicks`: number of heartbeat ticks for attempting to improve the mesh with opportunistic grafting.
  Default `60`
- `opportunisticGraftPeers`: the number of peers to opportunistically graft.
  Default `2`
- `graftFloodThresholdSeconds`: If a GRAFT comes before GraftFloodThresholdSeconds has elapsed since the last PRUNE, then there is an extra score penalty applied to the peer through P7. 
  Default `10` seconds
- `maxIHaveLength`: max number of messages to include in an IHAVE message, also controls the max number of IHAVE ids we will accept and request with IWANT from a peer within a heartbeat.
  Default `5000`
- `maxIHaveMessages`: max number of IHAVE messages to accept from a peer within a heartbeat.
  Default `10`
- `iWantFollowupTimeSeconds`: Time to wait for a message requested through IWANT following an IHAVE advertisement.
  Default `3` seconds
- `seenMessagesTTLSeconds`: configures when a previously seen message ID can be forgotten about.
  Default `120` seconds


### `Websocket` type

Type holding custom websocket support configuration:

```ts
interface Websocket {
    enabled?: bool;
    host?: string;
    port?: number;
    secure?: bool;
    certPath?: string;
    keyPath?: string;
}
```

Fields: 

All fields are optional.
If a key is `undefined`, or `null`, a default value will be set. If using `secure` websockets support, `certPath` and `keyPath` become mandatory attributes. Unless selfsigned certificates are used, it will probably make sense in the `JsonConfiguration` to specify the domain name used in the certificate in the `dns4DomainName` attribute.

- `enabled`:  indicates if websockets support will be enabled 
  Default `false`
- `host`: listening address for websocket connections
  Default `0.0.0.0`
- `port`: TCP listening port for websocket connection (`0` for random, binding to `443` requires root access)
  Default `60001`, if secure websockets support is enabled, the default is `6443“`
- `secure`: enable secure websockets support
  Default `false`
- `certPath`: secure websocket certificate path
- `keyPath`: secure websocket key path

### `extern void* waku_init()`

Allocate memory for a waku node. 

**Returns**

The result of this function must be passed to all waku_* functions that require a `void* ctx`

### `extern int waku_new(void* ctx, char* jsonConfig, WakuCallBack onErrCb void* userData)`

Instantiates a Waku node.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* jsonConfig`: JSON string containing the options used to initialize a go-waku node.
   Type [`JsonConfig`](#jsonconfig-type).
   It can be `NULL` to use defaults.
3. `WakuCallBack onErrCb`: callback to be executed if the function fails
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_start(void* ctx, WakuCallBack onErrCb void* userData)`

Start a Waku node mounting all the protocols that were enabled during the Waku node instantiation.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack onErrCb`: callback to be executed if the function fails
3. `void* userData`: used to pass custom information to the callback function


**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_stop(void* ctx, WakuCallBack onErrCb void* userData)`

Stops a Waku node.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack onErrCb`: callback to be executed if the function fails
3. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_free(void* ctx, WakuCallBack onErrCb void* userData)`

Release the resources allocated to a waku node (stopping the node first if necessary)

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack onErrCb`: callback to be executed if the function fails
3. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.


### `extern int waku_peerid(void* ctx, WakuCallBack cb, void* userData)`

Get the peer ID of the waku node.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack cb`: callback to be executed.
3. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the base58 encoded peer ID, for example `QmWjHKUrXDHPCwoWXpUZ77E8o6UbAoTTZwf1AD1tDC4KNP`.

### `extern int waku_listen_addresses(void* ctx, WakuCallBack cb, void* userData)`

Get the multiaddresses the Waku node is listening to.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack cb`: callback to be executed
3. `void* userData`: used to pass custom information to the callback function


**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive a json array of multiaddresses.
The multiaddresses are `string`s.
For example:
```json
[
    "/ip4/127.0.0.1/tcp/30303",
    "/ip4/1.2.3.4/tcp/30303",
    "/dns4/waku.node.example/tcp/8000/wss"
]
```


## Connecting to peers

### `extern int waku_add_peer(void* ctx, char* address, char* protocolId, WakuCallBack cb, void* userData)`

Add a node multiaddress and protocol to the waku node's peerstore.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* address`: A multiaddress (with peer id) to reach the peer being added.
3. `char* protocolId`: A protocol we expect the peer to support.
4. `WakuCallBack cb`: callback to be executed
5. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the base 58 peer ID of the peer that was added.
For example: `QmWjHKUrXDHPCwoWXpUZ77E8o6UbAoTTZwf1AD1tDC4KNP`

### `extern int waku_connect(void* ctx, char* address, int timeoutMs, WakuCallBack onErrCb void* userData)`

Dial peer using a multiaddress.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* address`: A multiaddress to reach the peer being dialed.
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
3. `WakuCallBack onErrCb`: callback to be executed if the function fails
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_connect_peerid(void* ctx, char* peerId, int timeoutMs, WakuCallBack onErrCb void* userData)`

Dial peer using its peer ID.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* peerID`: Peer ID to dial.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect`](#extern-char-waku_connectchar-address-int-timeoutms).
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
4. `WakuCallBack onErrCb`: callback to be executed if the function fails
5. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_disconnect(void* ctx, char* peerId, WakuCallBack onErrCb void* userData)`

Disconnect a peer using its peerID

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* peerID`: Peer ID to disconnect.
3. `WakuCallBack onErrCb`: callback to be executed if the function fails
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_peer_cnt(void* ctx, WakuCallBack cb, void* userData)`

Get number of connected peers.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack cb`: callback to be executed
3. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the number of connected peers.

### `extern int waku_peers(void* ctx, WakuCallBack cb, void* userData)`

Retrieve the list of peers known by the Waku node.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack cb`: callback to be executed if the function is succesful
3. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive a json array with the list of peers.
The list of peers has this format:

```json
[
  {
    "peerID": "16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47RedcBafeDCBA",
    "protocols": [
      "/ipfs/id/1.0.0",
      "/vac/waku/relay/2.0.0",
      "/ipfs/ping/1.0.0"
    ],
    "addrs": [
      "/ip4/1.2.3.4/tcp/30303"
    ],
    "connected": true
  }
]
```

## Waku Relay

### `extern int waku_content_topic(char* applicationName, unsigned int applicationVersion, char* contentTopicName, char* encoding, WakuCallBack onOkCb void* userData)`

Create a content topic string according to [RFC 23](https://rfc.vac.dev/spec/23/).

**Parameters**

1. `char* applicationName`
2. `unsigned int applicationVersion`
3. `char* contentTopicName`
4. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`
5. `WakuCallBack onOkCb`: callback to be executed if the function is succesful
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

If the function is executed succesfully, `onOkCb` will receive the content topic formatted according to [RFC 23](https://rfc.vac.dev/spec/23/).

```
/{application-name}/{version-of-the-application}/{content-topic-name}/{encoding}
```

### `extern int waku_pubsub_topic(char* name, char* encoding, WakuCallBack onOkCb, void* userData)`

Create a pubsub topic string according to [RFC 23](https://rfc.vac.dev/spec/23/).

**Parameters**

1. `char* name`
2. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`
3. `WakuCallBack onOkCb`: callback to be executed if the function is succesful
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

If the function is executed succesfully, `onOkCb` will receive the pubsub topic formatted according to [RFC 23](https://rfc.vac.dev/spec/23/).

```
/waku/2/{topic-name}/{encoding}
```

### `extern int waku_default_pubsub_topic(WakuCallBack onOkCb void* userData)`

Returns the default pubsub topic used for exchanging waku messages defined in [RFC 10](https://rfc.vac.dev/spec/10/).


**Parameters**

1. `char* name`
2. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`
3. `WakuCallBack onOkCb`: callback to be executed if the function is succesful
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

If the function is executed succesfully, `onOkCb` will receive the default pubsub topic: `/waku/2/default-waku/proto`

### `extern int waku_relay_publish(void* ctx, char* messageJson, char* pubsubTopic, int timeoutMs, WakuCallBack cb, void* userData)`

Publish a message using Waku Relay.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
3. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it derives the pubsub topic from content-topic based on autosharding.
4. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
5. `WakuCallBack cb`: callback to be executed
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `onOkCb` will receive the message ID.

### `extern int waku_relay_enough_peers(void* ctx, char* pubsubTopic, WakuCallBack cb, void* userData)`

Determine if there are enough peers to publish a message on a given pubsub topic.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* pubsubTopic`: Pubsub topic to verify.
   If `NULL`, it verifies the number of peers in the default pubsub topic.
3. `WakuCallBack cb`: callback to be executed
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `onOkCb` will receive a string `boolean` indicating whether there are enough peers, i.e. `true` or `false`

### `extern int waku_relay_subscribe(void* ctx, char* filterJSON, WakuCallBack onErrCb, void* userData)`

Subscribe to a Waku Relay pubsub topic to receive messages.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* filterJSON`: JSON string containing the [`ContentFilter`](#contentfilter-type) with the criteria of the messages to receive
3. `WakuCallBack onErrCb`: callback to be executed if the function fails
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

### `extern int waku_relay_topics(void* ctx, WakuCallBack cb, void* userData)`

Get the list of subscribed pubsub topics in Waku Relay.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `WakuCallBack cb`: callback to be executed
3. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive a json array of pubsub topics.

For example:

```json
["pubsubTopic1", "pubsubTopic2"]
```


**Events**

When a message is received, a `"message"` event` is emitted containing the message, pubsub topic, and node ID in which
the message was received.

The `event` type is [`JsonMessageEvent`](#jsonmessageevent-type).

For Example:

```json
{
  "type": "message",
  "event": {
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageID": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```

### `extern int waku_relay_unsubscribe(void* ctx, char* filterJSON, WakuCallBack onErrCb, void* userData)`

Closes the pubsub subscription to pubsub topic matching a criteria. No more messages will be received
from this pubsub topic.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* filterJSON`: JSON string containing the [`ContentFilter`](#contentfilter-type) with the criteria of the messages to unsubscribe from
3. `WakuCallBack onErrCb`: callback to be executed if the function fails
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.


## Waku Filter

### `extern int waku_filter_subscribe(void* ctx, char* filterJSON, char* peerID, int timeoutMs, WakuCallBack cb, void* userData)`

Creates a subscription to a filter full node matching a content filter..

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* filterJSON`: JSON string containing the [`ContentFilter`](#contentfilter-type) with the criteria of the messages to receive
3. `char* peerID`: Peer ID to subscribe to.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
   Use `NULL` to automatically select a node.
4. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
5. `WakuCallBack cb`: callback to be executed
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the following subscription details along with any partial errors.

For example:

```json
{
  "subscriptions" : [
    {
      "ID": "<subscriptionID>",
      "peerID": "....",
      "pubsubTopic": "...",
      "contentTopics": [...]
    }
  ],
  "error" : "subscriptions failed for contentTopics:<topicA>,.." // Empty if all subscriptions are succesful
}
```

**Events**

When a message is received, a ``"message"` event` is emitted containing the message, pubsub topic, and node ID in which
the message was received.

The `event` type is [`JsonMessageEvent`](#jsonmessageevent-type).

For Example:

```json
{
  "type": "message",
  "event": {
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageId": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```


### `extern int waku_filter_ping(void* ctx, char* peerID, int timeoutMs, WakuCallBack onErrCb, void* userData)`

Used to know if a service node has an active subscription for this client

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* peerID`: Peer ID to check for an active subscription
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
4. `WakuCallBack onErrCb`: callback to be executed if the function fails
5. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.


### `extern int waku_filter_unsubscribe(void* ctx, filterJSON *C.char, char* peerID, int timeoutMs, WakuCallBack onErrCb, void* userData)`

Sends a requests to a service node to stop pushing messages matching this filter to this client. It might be used to modify an existing subscription by providing a subset of the original filter criteria
**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* filterJSON`: JSON string containing the [`FilterSubscription`](#filtersubscription-type) criteria to unsubscribe from
3. `char* peerID`: Peer ID to unsubscribe from
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
4. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
5. `WakuCallBack onErrCb`: callback to be executed if the function fails
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.


### `extern int waku_filter_unsubscribe_all(void* ctx, char* peerID, int timeoutMs, WakuCallBack cb, void* userData)`

Sends a requests to a service node (or all service nodes) to stop pushing messages

**Parameters**

1. `char* peerID`: Peer ID to unsubscribe from
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
   Use `NULL` to unsubscribe from all peers with active subscriptions
2. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
3. `WakuCallBack cb`: callback to be executed
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive an array with information about the state of each unsubscription attempt (one per peer)

For example:

```json
[
  {
    "peerID": ....,
    "error": "" // Empty if succesful
  },
  ...
]
```


## Waku Legacy Filter

### `extern int waku_legacy_filter_subscribe(void* ctx, char* filterJSON, char* peerID, int timeoutMs, WakuCallBack onErrCb, void* userData)`

Creates a subscription in a lightnode for messages that matches a content filter and optionally a [PubSub `topic`](https://github.com/libp2p/specs/blob/master/pubsub/README.md#the-topic-descriptor).

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* filterJSON`: JSON string containing the [`LegacyFilterSubscription`](#legacyfiltersubscription-type) to subscribe to.
3. `char* peerID`: Peer ID to subscribe to.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
   Use `NULL` to automatically select a node.
4. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
5. `WakuCallBack onErrCb`: callback to be executed if the function fails
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

**Events**

When a message is received, a ``"message"` event` is emitted containing the message, pubsub topic, and node ID in which
the message was received.

The `event` type is [`JsonMessageEvent`](#jsonmessageevent-type).

For Example:

```json
{
  "type": "message",
  "event": {
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageId": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```

### `extern int waku_legacy_filter_unsubscribe(void* ctx, char* filterJSON, int timeoutMs, WakuCallBack onErrCb, void* userData)`

Removes subscriptions in a light node matching a content filter and, optionally, a [PubSub `topic`](https://github.com/libp2p/specs/blob/master/pubsub/README.md#the-topic-descriptor).

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* filterJSON`: JSON string containing the [`LegacyFilterSubscription`](#filtersubscription-type).
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
4. `WakuCallBack onErrCb`: callback to be executed if the function fails
5. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.

## Waku Lightpush

### `extern int waku_lightpush_publish(void* ctx, char* messageJSON, char* topic, char* peerID, int timeoutMs, WakuCallBack cb, void* userData)`

Publish a message using Waku Lightpush.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
3. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it derives the pubsub topic from content-topic based on autosharding.
4. `char* peerID`: Peer ID supporting the lightpush protocol.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
   Use `NULL` to automatically select a node.
5. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
6. `WakuCallBack cb`: callback to be executed
7. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the message ID.


## Waku Store

### `extern int waku_store_query(void* ctx, char* queryJSON, char* peerID, int timeoutMs, WakuCallBack cb, void* userData)`

Retrieves historical messages on specific content topics. This method may be called with [`PagingOptions`](#pagingoptions-type), 
to retrieve historical messages on a per-page basis. If the request included [`PagingOptions`](#pagingoptions-type), the node 
must return messages on a per-page basis and include [`PagingOptions`](#pagingoptions-type) in the response. These [`PagingOptions`](#pagingoptions-type) 
must contain a cursor pointing to the Index from which a new page can be requested.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* queryJSON`: JSON string containing the [`StoreQuery`](#storequery-type).
3. `char* peerID`: Peer ID supporting the store protocol.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
4. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
5. `WakuCallBack cb`: callback to be executed
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive a [`StoreResponse`](#storeresponse-type).

### `extern int waku_store_local_query(void* ctx, char* queryJSON, WakuCallBack cb, void* userData)`

Retrieves locally stored historical messages on specific content topics. This method may be called with [`PagingOptions`](#pagingoptions-type), 
to retrieve historical messages on a per-page basis. If the request included [`PagingOptions`](#pagingoptions-type), the node 
must return messages on a per-page basis and include [`PagingOptions`](#pagingoptions-type) in the response. These [`PagingOptions`](#pagingoptions-type) 
must contain a cursor pointing to the Index from which a new page can be requested.

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* queryJSON`: JSON string containing the [`StoreQuery`](#storequery-type).
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
4. `WakuCallBack cb`: callback to be executed
5. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive a [`StoreResponse`](#storeresponse-type).

## Encrypting messages

### `extern int waku_encode_symmetric(char* messageJson, char* symmetricKey, char* optionalSigningKey, WakuCallBack cb, void* userData)`

Encrypt a message using symmetric encryption and optionally sign the message

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* symmetricKey`: hex encoded secret key to be used for encryption.
3. `char* optionalSigningKey`: hex encoded private key to be used to sign the message.
4. `WakuCallBack cb`: callback to be executed
5. `void* userData`: used to pass custom information to the callback function

Note: `messageJson.version` is overwritten to `1`.

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the encrypted waku message which can be broadcasted with relay or lightpush protocol publish functions

### `extern int waku_encode_asymmetric(char* messageJson, char* publicKey, char* optionalSigningKey, WakuCallBack cb, void* userData)`

Encrypt a message using asymmetric encryption and optionally sign the message

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* publicKey`: hex encoded public key to be used for encryption.
3. `char* optionalSigningKey`: hex encoded private key to be used to sign the message.
4. `WakuCallBack cb`: callback to be executed
5. `void* userData`: used to pass custom information to the callback function

Note: `messageJson.version` is overwritten to `1`.

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the encrypted waku message which can be broadcasted with relay or lightpush protocol publish functions


## Decrypting messages

### `extern int waku_decode_symmetric(char* messageJson, char* symmetricKey, WakuCallBack cb, void* userData)`
Decrypt a message using a symmetric key

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* symmetricKey`: 32 byte symmetric key hex encoded.
3. `WakuCallBack cb`: callback to be executed
4. `void* userData`: used to pass custom information to the callback function

Note: `messageJson.version` is expected to be `1`.

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the decoded payload as a [`DecodedPayload`](#decodedpayload-type).


```json
{
  "pubkey": "0x......",
  "signature": "0x....",
  "data": "...",
  "padding": "..."
}
```

### `extern int waku_decode_asymmetric(char* messageJson, char* privateKey, WakuCallBack cb, void* userData)`
Decrypt a message using a secp256k1 private key 

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* privateKey`: secp256k1 private key hex encoded.
3. `WakuCallBack cb`: callback to be executed
4. `void* userData`: used to pass custom information to the callback function

Note: `messageJson.version` is expected to be `1`.

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `cb` will receive the decoded payload as a [`DecodedPayload`](#decodedpayload-type).

```json
{
  "pubkey": "0x......",
  "signature": "0x....",
  "data": "...",
  "padding": "..."
}
```

## DNS Discovery

### `extern int waku_dns_discovery(void* ctx, char* url, char* nameserver, int timeoutMs, WakuCallBack cb, void* userData)`
Returns a list of multiaddress and enrs given a url to a DNS discoverable ENR tree

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* url`: URL containing a discoverable ENR tree
3. `char* nameserver`: The nameserver to resolve the ENR tree url. 
   If `NULL` or empty, it will automatically use the default system dns.
4. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.
5. `WakuCallBack cb`: callback to be executed
6. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values.
If the function execution fails, `cb` will receive a string containing an error.
If the function is executed succesfully, `onOkCb` will receive an array objects describing the multiaddresses, enr and peerID each node found.

```json
[
    {
        "peerID":"16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W",
        "multiaddrs":[
            "/dns4/node-01.do-ams3.waku.test.status.im/tcp/30303/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W",
            "/dns4/node-01.do-ams3.waku.test.status.im/tcp/8000/wss/p2p/16Uiu2HAkykgaECHswi3YKJ5dMLbq2kPVCo89fcyTd38UcQD6ej5W"
        ],
        "enr":"enr:-QESuEC1p_s3xJzAC_XlOuuNrhVUETmfhbm1wxRGis0f7DlqGSw2FM-p2Ugl_r25UHQJ3f1rIRrpzxJXSMaJe4yk1XFSAYJpZIJ2NIJpcISygI2rim11bHRpYWRkcnO4XAArNiZub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1c2ltLm5ldAZ2XwAtNiZub2RlLTAxLmRvLWFtczMud2FrdS50ZXN0LnN0YXR1c2ltLm5ldAYfQN4DgnJzkwABCAAAAAEAAgADAAQABQAGAAeJc2VjcDI1NmsxoQJATXRSRSUyTw_QLB6H_U3oziVQgNRgrXpK7wp2AMyNxYN0Y3CCdl-DdWRwgiMohXdha3UyDw"
    },
    ...
]
```

## DiscoveryV5

### `extern int waku_discv5_update_bootnodes(void* ctx, char* bootnodes, WakuCallBack onErrCb, void* userData)`
Update the bootnode list used for discovering new peers via DiscoveryV5

**Parameters**

1. `void* ctx`: waku node instance, returned by `waku_init`.
2. `char* bootnodes`: JSON array containing the bootnode ENRs i.e. `["enr:...", "enr:..."]`
3. `WakuCallBack onErrCb`: callback to be executed if the function fails
4. `void* userData`: used to pass custom information to the callback function

**Returns**

A status code. Refer to the [`Status codes`](#status-codes) section for possible values


# Copyright

Copyright and related rights waived via
[CC0](https://creativecommons.org/publicdomain/zero/1.0/).
