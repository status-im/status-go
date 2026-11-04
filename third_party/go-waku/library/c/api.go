// Set of functions that are exported when go-waku is built as a static/dynamic library
package main

/*
#include <stdlib.h>
#include <stddef.h>

// The possible returned values for the functions that return int
static const int RET_OK = 0;
static const int RET_ERR = 1;
static const int RET_MISSING_CALLBACK = 2;

typedef void (*WakuCallBack) (int ret_code, const char* msg, void * user_data);
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/waku-org/go-waku/library"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

func main() {}

// Initialize a waku node. Receives a JSON string containing the configuration
// for the node. It can be NULL. Example configuration:
// ```
// {"host": "0.0.0.0", "port": 60000, "advertiseAddr": "1.2.3.4", "nodeKey": "0x123...567", "keepAliveInterval": 20, "relay": true}
// ```
// All keys are optional. If not specified a default value will be set:
// - host: IP address. Default 0.0.0.0
// - port: TCP port to listen. Default 60000. Use 0 for random
// - advertiseAddr: External IP
// - nodeKey: secp256k1 private key. Default random
// - keepAliveInterval: interval in seconds to ping all peers
// - relay: Enable WakuRelay. Default `true`
// - relayTopics: Array of pubsub topics that WakuRelay will automatically subscribe to when the node starts
// - gossipsubParams: an object containing custom gossipsub parameters. All attributes are optional, and if not specified, it will use default values.
//   - d: optimal degree for a GossipSub topic mesh. Default `6`
//   - dLow: lower bound on the number of peers we keep in a GossipSub topic mesh. Default `5`
//   - dHigh: upper bound on the number of peers we keep in a GossipSub topic mesh. Default `12`
//   - dScore: affects how peers are selected when pruning a mesh due to over subscription. Default `4`
//   - dOut: sets the quota for the number of outbound connections to maintain in a topic mesh. Default `2`
//   - historyLength: controls the size of the message cache used for gossip. Default `5`
//   - historyGossip: controls how many cached message ids we will advertise in IHAVE gossip messages. Default `3`
//   - dLazy: affects how many peers we will emit gossip to at each heartbeat. Default `6`
//   - gossipFactor: affects how many peers we will emit gossip to at each heartbeat. Default `0.25`
//   - gossipRetransmission: controls how many times we will allow a peer to request the same message id through IWANT gossip before we start ignoring them. Default `3`
//   - heartbeatInitialDelayMs: short delay in milliseconds before the heartbeat timer begins after the router is initialized. Default `100` milliseconds
//   - heartbeatIntervalSeconds: controls the time between heartbeats. Default `1` second
//   - slowHeartbeatWarning: duration threshold for heartbeat processing before emitting a warning. Default `0.1`
//   - fanoutTTLSeconds: controls how long we keep track of the fanout state. Default `60` seconds
//   - prunePeers: controls the number of peers to include in prune Peer eXchange. Default `16`
//   - pruneBackoffSeconds: controls the backoff time for pruned peers. Default `60` seconds
//   - unsubscribeBackoffSeconds: controls the backoff time to use when unsuscribing from a topic. Default `10` seconds
//   - connectors: number of active connection attempts for peers obtained through PX. Default `8`
//   - maxPendingConnections: maximum number of pending connections for peers attempted through px. Default `128`
//   - connectionTimeoutSeconds: timeout in seconds for connection attempts. Default `30` seconds
//   - directConnectTicks: the number of heartbeat ticks for attempting to reconnect direct peers that are not currently connected. Default `300`
//   - directConnectInitialDelaySeconds: initial delay before opening connections to direct peers. Default `1` second
//   - opportunisticGraftTicks: number of heartbeat ticks for attempting to improve the mesh with opportunistic grafting. Default `60`
//   - opportunisticGraftPeers: the number of peers to opportunistically graft. Default `2`
//   - graftFloodThresholdSeconds: If a GRAFT comes before GraftFloodThresholdSeconds has elapsed since the last PRUNE, then there is an extra score penalty applied to the peer through P7. Default `10` seconds
//   - maxIHaveLength: max number of messages to include in an IHAVE message, also controls the max number of IHAVE ids we will accept and request with IWANT from a peer within a heartbeat. Default `5000`
//   - maxIHaveMessages: max number of IHAVE messages to accept from a peer within a heartbeat. Default `10`
//   - iWantFollowupTimeSeconds: Time to wait for a message requested through IWANT following an IHAVE advertisement. Default `3` seconds
//   - seenMessagesTTLSeconds:  configures when a previously seen message ID can be forgotten about. Default `120` seconds
//
// - minPeersToPublish: The minimum number of peers required on a topic to allow broadcasting a message. Default `0`
// - legacyFilter: Enable LegacyFilter. Default `false`
// - discV5: Enable DiscoveryV5. Default `false`
// - discV5BootstrapNodes: Array of bootstrap nodes ENR
// - discV5UDPPort: UDP port for DiscoveryV5
// - logLevel: Set the log level. Default `INFO`. Allowed values "DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL"
// - store: Enable Store. Default `false`
// - databaseURL: url connection string. Default: "sqlite3://store.db". Also accepts PostgreSQL connection strings
// - storeRetentionMaxMessages: max number of messages to store in the database. Default 10000
// - storeRetentionTimeSeconds: max number of seconds that a message will be persisted in the database. Default 2592000 (30d)
// - websockets: an optional object containing settings to setup the websocket configuration
//   - enabled: indicates if websockets support will be enabled. Default `false`
//   - host: listening address for websocket connections. Default `0.0.0.0`
//   - port: TCP listening port for websocket connection (0 for random, binding to 443 requires root access). Default: `60001“, if secure websockets support is enabled, the default is `6443“
//   - secure: enable secure websockets support. Default `false`
//   - certPath: secure websocket certificate path
//   - keyPath: secure websocket key path
//
// - dns4DomainName: the domain name resolving to the node's public IPv4 address.
//
//export waku_new
func waku_new(configJSON *C.char, cb C.WakuCallBack, userData unsafe.Pointer) unsafe.Pointer {
	if cb == nil {
		panic("error: missing callback in waku_new")
	}

	cid := C.malloc(C.size_t(unsafe.Sizeof(uintptr(0))))
	pid := (*uint)(cid)
	instance := library.Init()
	*pid = instance.ID

	err := library.NewNode(instance, C.GoString(configJSON))
	if err != nil {
		onError(err, cb, userData)
		return nil
	}

	return cid
}

// Starts the waku node
//
//export waku_start
func waku_start(ctx unsafe.Pointer, onErr C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, onErr, userData)
	}

	err = library.Start(instance)
	return onError(err, onErr, userData)
}

// Stops a waku node
//
//export waku_stop
func waku_stop(ctx unsafe.Pointer, onErr C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, onErr, userData)
	}

	err = library.Stop(instance)
	return onError(err, onErr, userData)
}

// Release the resources allocated to a waku node (stopping the node first if necessary)
//
//export waku_free
func waku_free(ctx unsafe.Pointer, onErr C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		return onError(err, onErr, userData)
	}

	err = library.Stop(instance)
	if err != nil {
		return onError(err, onErr, userData)
	}

	err = library.Free(instance)
	if err == nil {
		C.free(ctx)
	}

	return onError(err, onErr, userData)
}

// Determine is a node is started or not
//
//export waku_is_started
func waku_is_started(ctx unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		return 0
	}

	started := library.IsStarted(instance)
	if started {
		return 1
	}
	return 0
}

type fn func(instance *library.WakuInstance) (string, error)
type fnNoInstance func() (string, error)

func singleFnExec(f fn, ctx unsafe.Pointer, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	result, err := f(instance)
	if err != nil {
		return onError(err, cb, userData)
	}
	return onSuccesfulResponse(result, cb, userData)
}

func singleFnExecNoCtx(f fnNoInstance, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	result, err := f()
	if err != nil {
		return onError(err, cb, userData)
	}
	return onSuccesfulResponse(result, cb, userData)
}

// Obtain the peer ID of the waku node
//
//export waku_peerid
func waku_peerid(ctx unsafe.Pointer, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.PeerID(instance)
	}, ctx, cb, userData)
}

// Obtain the multiaddresses the wakunode is listening to
//
//export waku_listen_addresses
func waku_listen_addresses(ctx unsafe.Pointer, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.ListenAddresses(instance)
	}, ctx, cb, userData)
}

// Add node multiaddress and protocol to the wakunode peerstore
//
//export waku_add_peer
func waku_add_peer(ctx unsafe.Pointer, address *C.char, protocolID *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.AddPeer(instance, C.GoString(address), C.GoString(protocolID))
	}, ctx, cb, userData)
}

// Connect to peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
//
//export waku_connect
func waku_connect(ctx unsafe.Pointer, address *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.Connect(instance, C.GoString(address), int(ms))
	return onError(err, cb, userData)
}

// Connect to known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
//
//export waku_connect_peerid
func waku_connect_peerid(ctx unsafe.Pointer, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.ConnectPeerID(instance, C.GoString(peerID), int(ms))
	return onError(err, cb, userData)
}

// Close connection to a known peer by peerID
//
//export waku_disconnect
func waku_disconnect(ctx unsafe.Pointer, peerID *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.Disconnect(instance, C.GoString(peerID))
	return onError(err, cb, userData)
}

// Get number of connected peers
//
//export waku_peer_cnt
func waku_peer_cnt(ctx unsafe.Pointer, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		peerCnt, err := library.PeerCnt(instance)
		return fmt.Sprintf("%d", peerCnt), err
	}, ctx, cb, userData)
}

// Create a content topic string according to RFC 23
//
//export waku_content_topic
func waku_content_topic(applicationName *C.char, applicationVersion *C.char, contentTopicName *C.char, encoding *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	contentTopic, _ := protocol.NewContentTopic(C.GoString(applicationName), C.GoString(applicationVersion), C.GoString(contentTopicName), C.GoString(encoding))
	return onSuccesfulResponse(contentTopic.String(), cb, userData)
}

// Get the default pubsub topic used in waku2: /waku/2/default-waku/proto
//
//export waku_default_pubsub_topic
func waku_default_pubsub_topic(cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return onSuccesfulResponse(library.DefaultPubsubTopic(), cb, userData)
}

// Register callback to act as signal handler and receive application signals
// (in JSON) which are used to react to asynchronous events in waku. The function
// signature for the callback should be `void myCallback(char* signalJSON)`
//
//export waku_set_event_callback
func waku_set_event_callback(ctx unsafe.Pointer, cb C.WakuCallBack) {
	instance, err := getInstance(ctx)
	if err != nil {
		panic(err.Error()) // TODO: refactor to return an error instead of panic
	}

	library.SetEventCallback(instance, unsafe.Pointer(cb))
}

// Retrieve the list of peers known by the waku node
//
//export waku_peers
func waku_peers(ctx unsafe.Pointer, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.Peers(instance)
	}, ctx, cb, userData)
}
