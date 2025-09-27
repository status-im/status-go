package types

import "crypto/ecdsa"

type Persistence interface {
	wakuPersistence

	MessageCacheAdd(ids []string, timestamp uint64) error
	MessageCacheClear() error
	MessageCacheClearOlderThan(timestamp uint64) error
	MessageCacheHits(ids []string) (map[string]bool, error)

	InsertPendingConfirmation(confirmation *RawMessageConfirmation) error
	SaveHashRatchetMessage(groupID []byte, keyID []byte, m *ReceivedMessage) error
	GetHashRatchetMessages(keyID []byte) ([]*ReceivedMessage, error)
	DeleteHashRatchetMessages(ids [][]byte) error
	DeleteHashRatchetMessagesOlderThan(timestamp int64) error

	IsMessageAlreadyCompleted(hash []byte) (bool, error)
	SaveMessageSegment(segment *SegmentMessage, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*SegmentMessage, error)
	CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	RemoveMessageSegmentsOlderThan(timestamp int64) error
	RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error
}

type ProtectedTopic struct {
	PubKey *ecdsa.PublicKey
	Topic  string
}

type wakuPersistence interface {
	WakuKeys() (map[string][]byte, error)
	AddWakuKey(chatID string, key []byte) error
	WakuInsertProtectedTopic(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error
	WakuDeleteProtectedTopic(pubsubTopic string) error
	WakuFetchPrivateKeyForProtectedTopic(topic string) (*ecdsa.PrivateKey, error)
	WakuProtectedTopics() ([]ProtectedTopic, error)
}
