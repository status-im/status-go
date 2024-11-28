package types

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/pborman/uuid"

	"github.com/waku-org/go-waku/waku/v2/api/history"

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

	// PeerID returns node's PeerID
	PeerID() peer.ID

	// GetActiveStorenode returns the peer ID of the currently active storenode. It will be empty if no storenode is active
	GetActiveStorenode() peer.ID

	// OnStorenodeChanged is triggered when a new storenode is promoted to become the active storenode or when the active storenode is removed
	OnStorenodeChanged() <-chan peer.ID

	// OnStorenodeNotWorking is triggered when the last active storenode fails to return results consistently
	OnStorenodeNotWorking() <-chan struct{}

	// OnStorenodeAvailable is triggered when there is a new active storenode selected
	OnStorenodeAvailable() <-chan peer.ID

	// WaitForAvailableStoreNode will wait for a storenode to be available depending on the context
	WaitForAvailableStoreNode(ctx context.Context) bool

	// SetStorenodeConfigProvider will set the configuration provider for the storenode cycle
	SetStorenodeConfigProvider(c history.StorenodeConfigProvider)

	// ProcessMailserverBatch will receive a criteria and storenode and execute a query
	ProcessMailserverBatch(
		ctx context.Context,
		batch MailserverBatch,
		storenodeID peer.ID,
		pageLimit uint64,
		shouldProcessNextPage func(int) (bool, uint64),
		processEnvelopes bool,
	) error

	// IsStorenodeAvailable is used to determine whether a storenode is available or not
	IsStorenodeAvailable(peerID peer.ID) bool

	PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error

	// DisconnectActiveStorenode will trigger a disconnection of the active storenode, and potentially execute a cycling so a new storenode is promoted
	DisconnectActiveStorenode(ctx context.Context, backoff time.Duration, shouldCycle bool)
}

type MailserverBatch struct {
	From        time.Time
	To          time.Time
	Cursor      string
	PubsubTopic string
	Topics      []TopicType
	ChatIDs     []string
}

func (mb *MailserverBatch) Hash() string {
	data := fmt.Sprintf("%d%d%s%s%v%v", mb.From.UnixNano(), mb.To.UnixNano(), mb.Cursor, mb.PubsubTopic, mb.Topics, mb.ChatIDs)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:4])
}
