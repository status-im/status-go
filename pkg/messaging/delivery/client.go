package delivery

import (
	"context"
	"crypto/ecdsa"

	"github.com/libp2p/go-libp2p/core/peer"

	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// Client is the part of a logos-delivery Messaging API client the Adapter
// drives. NewClient returns the one backed by liblogosdelivery.
type Client interface {
	Start() error
	Stop() error
	// Close releases the node and closes the Events channel.
	Close() error
	Events() <-chan Event
	Subscribe(contentTopic string) error
	Unsubscribe(contentTopic string) error
	// Send queues payload on contentTopic and returns the request ID that
	// correlates it with the SentEvent, PropagatedEvent or ErrorEvent it produces.
	Send(ctx context.Context, contentTopic string, payload []byte, ephemeral bool) (string, error)

	// StoreQuery runs one store request against peerInfo and returns the page
	// of messages and the cursor of the next page, nil on the last one.
	StoreQuery(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error)
	// ConnectedPeers returns the protocols and addresses of every connected peer.
	ConnectedPeers() (types.PeerStats, error)
	PeerID() (peer.ID, error)
	// Metrics returns the node's metrics in the Prometheus text format.
	Metrics() (string, error)
}

// Event is one Messaging API event: ReceivedEvent, SentEvent, PropagatedEvent,
// ErrorEvent or ConnectionEvent.
type Event interface {
	isDeliveryEvent()
}

// ReceivedEvent is a message delivered on a subscribed content topic.
type ReceivedEvent struct {
	// MessageHash is the 0x-prefixed hex wire hash.
	MessageHash  string
	ContentTopic string
	Payload      []byte
	Meta         []byte
	Version      uint32
	// Timestamp is sender-generated, in nanoseconds.
	Timestamp int64
	Ephemeral bool
}

// SentEvent reports a send that a store node confirmed. It follows the
// PropagatedEvent, and only when the node validates sends with store.
type SentEvent struct {
	RequestID   string
	MessageHash string
}

// PropagatedEvent reports a message that reached the network.
type PropagatedEvent struct {
	RequestID   string
	MessageHash string
}

// ErrorEvent reports a send that failed.
type ErrorEvent struct {
	RequestID   string
	MessageHash string
	Err         string
}

// ConnectionStatus mirrors the Messaging API's three-state connection status.
type ConnectionStatus int

const (
	Disconnected ConnectionStatus = iota
	PartiallyConnected
	Connected
)

// ConnectionEvent reports a change of the node's connection status.
type ConnectionEvent struct {
	Status ConnectionStatus
}

func (ReceivedEvent) isDeliveryEvent()   {}
func (SentEvent) isDeliveryEvent()       {}
func (PropagatedEvent) isDeliveryEvent() {}
func (ErrorEvent) isDeliveryEvent()      {}
func (ConnectionEvent) isDeliveryEvent() {}

// Mode is the node role.
type Mode string

const (
	ModeCore Mode = "Core"
	ModeEdge Mode = "Edge"
)

// Config configures the node NewClient creates.
type Config struct {
	// Preset is the logos-delivery network preset, e.g. "status.prod".
	Preset string
	Mode   Mode
	// NodeKey is the libp2p identity key. A random one is used when nil.
	NodeKey *ecdsa.PrivateKey
	// TCPPort and UDPPort are the libp2p and discv5 ports; zero keeps the
	// library default.
	TCPPort int
	UDPPort int
	// DataDir is where the node keeps its message store. It must not be shared
	// with another node.
	DataDir string
	// Overrides are extra node configuration fields, keyed by field or CLI
	// switch name. They take precedence over the fields above.
	Overrides map[string]any
}
