package whispertypes

import (
	"crypto/ecdsa"
	"time"
)

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Whisper interface {
	PublicWhisperAPI() PublicWhisperAPI
	NewMessageStore() MessageStore

	// MinPow returns the PoW value required by this node.
	MinPow() float64
	// BloomFilter returns the aggregated bloom filter for all the topics of interest.
	// The nodes are required to send only messages that match the advertised bloom filter.
	// If a message does not match the bloom, it will tantamount to spam, and the peer will
	// be disconnected.
	BloomFilter() []byte
	// SetTimeSource assigns a particular source of time to a whisper object.
	SetTimeSource(timesource func() time.Time)
	// GetCurrentTime returns current time.
	GetCurrentTime() time.Time

	// SelectedKeyPairID returns the id of currently selected key pair.
	// It helps distinguish between different users w/o exposing the user identity itself.
	SelectedKeyPairID() string
	// GetPrivateKey retrieves the private key of the specified identity.
	GetPrivateKey(id string) (*ecdsa.PrivateKey, error)

	SubscribeEnvelopeEvents(events chan<- EnvelopeEvent) Subscription

	// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
	AddKeyPair(key *ecdsa.PrivateKey) (string, error)
	// DeleteKeyPair deletes the specified key if it exists.
	DeleteKeyPair(key string) bool
	// SelectKeyPair adds cryptographic identity, and makes sure
	// that it is the only private key known to the node.
	SelectKeyPair(key *ecdsa.PrivateKey) error
	AddSymKeyDirect(key []byte) (string, error)
	AddSymKeyFromPassword(password string) (string, error)
	DeleteSymKey(id string) bool
	GetSymKey(id string) ([]byte, error)

	Subscribe(f Filter) (string, error)
	GetFilter(id string) Filter
	Unsubscribe(id string) error

	CreateFilterWrapper(keyAsym *ecdsa.PrivateKey, keySym []byte, pow float64, topics [][]byte, messages MessageStore) Filter

	// RequestHistoricMessages sends a message with p2pRequestCode to a specific peer,
	// which is known to implement MailServer interface, and is supposed to process this
	// request and respond with a number of peer-to-peer messages (possibly expired),
	// which are not supposed to be forwarded any further.
	// The whisper protocol is agnostic of the format and contents of envelope.
	// A timeout of 0 never expires.
	RequestHistoricMessagesWithTimeout(peerID []byte, envelope Envelope, timeout time.Duration) error
	// SendMessagesRequest sends a MessagesRequest. This is an equivalent to RequestHistoricMessages
	// in terms of the functionality.
	SendMessagesRequest(peerID []byte, request MessagesRequest) error
	// SyncMessages can be sent between two Mail Servers and syncs envelopes between them.
	SyncMessages(peerID []byte, req SyncMailRequest) error
}
