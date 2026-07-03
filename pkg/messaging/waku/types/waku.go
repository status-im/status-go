package types

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/pborman/uuid"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/connection"
)

type ConnStatus struct {
	IsOnline bool `json:"isOnline"`
}

type PeerStats map[peer.ID]WakuV2Peer

func (m PeerStats) MarshalJSON() ([]byte, error) {
	tmpMap := make(map[string]WakuV2Peer)
	for k, v := range m {
		tmpMap[k.String()] = v
	}
	return json.Marshal(tmpMap)
}

type WakuV2Peer struct {
	Protocols []protocol.ID         `json:"protocols"`
	Addresses []multiaddr.Multiaddr `json:"addresses"`
}

type ConnStatusSubscription struct {
	sync.RWMutex

	ID     string
	C      chan ConnStatus
	active bool
}

func NewConnStatusSubscription() *ConnStatusSubscription {
	return &ConnStatusSubscription{
		ID:     uuid.NewRandom().String(),
		C:      make(chan ConnStatus, 100),
		active: true,
	}
}

func (u *ConnStatusSubscription) Active() bool {
	u.RLock()
	defer u.RUnlock()
	return u.active
}

func (u *ConnStatusSubscription) Unsubscribe() {
	u.Lock()
	defer u.Unlock()
	close(u.C)
	u.active = false
}

func (u *ConnStatusSubscription) Send(s ConnStatus) bool {
	u.RLock()
	defer u.RUnlock()
	if !u.active {
		return false
	}
	u.C <- s
	return true
}

// TopicSubscription identifies a single (pubsub topic, content topic) pair the
// transport wants to receive messages on. It is the unit of wire-subscription
// bookkeeping, both in the transport's FilterSubscriptions and in the waku
// adapter.
type TopicSubscription struct {
	PubsubTopic  string
	ContentTopic TopicType
}

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku interface {
	// Send publishes a pre-encoded payload on the messaging network. The
	// payload is expected to be already encoded for WakuMessage version=1
	// (see transport/rfc26.Encode). Returns the wire hash on success.
	Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error)

	Start() error
	Stop() error

	// Pause signals the waku transport to idle its goroutines (e.g. when the
	// hosting app is backgrounded). Idempotent. Cascaded from the messenger
	// transport so the protocol layer can suppress background work.
	Pause() error
	// Resume re-arms goroutines suspended by Pause. Idempotent.
	Resume() error

	// Peers is retained only for the Python functional tests (see tests-functional);
	// it is not used by status-app.
	Peers() PeerStats

	SubscribeToConnStatusChanges() (*ConnStatusSubscription, error)

	SubscribeEnvelopeEvents(events chan<- EnvelopeEvent) Subscription

	MaxMessageSize() uint32

	// ConnectionChanged is called whenever the client knows its connection status has changed
	ConnectionChanged(connection.State)

	// ConfirmMessageDelivered updates a message has been delivered in waku
	ConfirmMessageDelivered(hash []common.Hash)

	// PeerID returns node's PeerID
	PeerID() peer.ID

	// StoreQuery retrieves historic messages for a single batch, selecting the
	// store node internally (no peer argument). See waku.StoreClient.
	StoreQuery(
		ctx context.Context,
		batch MailserverBatch,
		pageLimit uint64,
		shouldProcessNextPage func(int) (bool, uint64),
		processEnvelopes bool,
	) error
}

type MailserverBatch struct {
	From        time.Time
	To          time.Time
	PubsubTopic string
	Topics      []TopicType
	ChatIDs     []string
}
