package types

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/pborman/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/connection"
)

type ConnStatus struct {
	IsOnline bool      `json:"isOnline"`
	Peers    PeerStats `json:"peers"`
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

type PeerList struct {
	FullMeshPeers peer.IDSlice `json:"fullMesh"`
	AllPeers      peer.IDSlice `json:"all"`
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

type WakuKeyManager interface {
	// GetPrivateKey retrieves the private key of the specified identity.
	GetPrivateKey(id string) (*ecdsa.PrivateKey, error)
	// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
	AddKeyPair(key *ecdsa.PrivateKey) (string, error)
	// DeleteKeyPair deletes the key with the specified ID if it exists.
	DeleteKeyPair(keyID string) bool
	// DeleteKeyPairs deletes all the keys
	DeleteKeyPairs() error
	AddSymKeyDirect(key []byte) (string, error)
	AddSymKeyFromPassword(password string) (string, error)
	DeleteSymKey(id string) bool
	GetSymKey(id string) ([]byte, error)
}

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Waku interface {
	PublicWakuAPI() PublicWakuAPI

	// Waku protocol version
	Version() uint

	// PeerCount
	PeerCount() int

	ListenAddresses() ([]multiaddr.Multiaddr, error)

	RelayPeersByTopic(topic string) (*PeerList, error)

	ENR() (*enode.Node, error)

	Peers() PeerStats

	StartDiscV5() error

	StopDiscV5() error

	SubscribeToPubsubTopic(topic string, optPublicKey *ecdsa.PublicKey) error

	UnsubscribeFromPubsubTopic(topic string) error

	StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error

	RetrievePubsubTopicKey(topic string) (*ecdsa.PrivateKey, error)

	RemovePubsubTopicKey(topic string) error

	AddStorePeer(address multiaddr.Multiaddr) (peer.ID, error)

	AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error)

	DialPeer(address multiaddr.Multiaddr) error

	DialPeerByID(peerID peer.ID) error

	DropPeer(peerID peer.ID) error

	SubscribeToConnStatusChanges() (*ConnStatusSubscription, error)

	SetCriteriaForMissingMessageVerification(peerID peer.ID, pubsubTopic string, contentTopics []TopicType) error

	// MinPow returns the PoW value required by this node.
	MinPow() float64
	// BloomFilter returns the aggregated bloom filter for all the topics of interest.
	// The nodes are required to send only messages that match the advertised bloom filter.
	// If a message does not match the bloom, it will tantamount to spam, and the peer will
	// be disconnected.
	BloomFilter() []byte

	// GetCurrentTime returns current time.
	GetCurrentTime() time.Time

	// GetPrivateKey retrieves the private key of the specified identity.
	GetPrivateKey(id string) (*ecdsa.PrivateKey, error)

	SubscribeEnvelopeEvents(events chan<- EnvelopeEvent) Subscription

	// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
	AddKeyPair(key *ecdsa.PrivateKey) (string, error)
	// DeleteKeyPair deletes the key with the specified ID if it exists.
	DeleteKeyPair(keyID string) bool
	AddSymKeyDirect(key []byte) (string, error)
	AddSymKeyFromPassword(password string) (string, error)
	DeleteSymKey(id string) bool
	GetSymKey(id string) ([]byte, error)
	MaxMessageSize() uint32

	GetStats() StatsSummary

	Subscribe(opts *SubscriptionOptions) (string, error)
	GetFilter(id string) Filter
	Unsubscribe(ctx context.Context, id string) error
	UnsubscribeMany(ids []string) error

	// RequestStoreMessages uses the WAKU2-STORE protocol to request historic messages
	RequestStoreMessages(ctx context.Context, peerID peer.ID, request MessagesRequest, processEnvelopes bool) (StoreRequestCursor, int, error)

	// ProcessingP2PMessages indicates whether there are in-flight p2p messages
	ProcessingP2PMessages() bool

	// MarkP2PMessageAsProcessed tells the waku layer that a P2P message has been processed
	MarkP2PMessageAsProcessed(common.Hash)

	// ConnectionChanged is called whenever the client knows its connection status has changed
	ConnectionChanged(connection.State)

	// ClearEnvelopesCache clears waku envelopes cache
	ClearEnvelopesCache()

	// ConfirmMessageDelivered updates a message has been delivered in waku
	ConfirmMessageDelivered(hash []common.Hash)

	// SetStorePeerID updates the peer id of store node
	SetStorePeerID(peerID peer.ID)

	// PeerID returns node's PeerID
	PeerID() peer.ID

	// PingPeer returns the reply time
	PingPeer(ctx context.Context, peerID peer.ID) (time.Duration, error)
}
