package types

import (
	"crypto/ecdsa"
)

type Persistence interface {
	WakuStorage() WakuPersistence
	SegmentationStorage() SegmentationPersistence
	EncryptionStorage() EncryptionPersistence

	MessageCacheAdd(ids []string, timestamp uint64) error
	MessageCacheClear() error
	MessageCacheClearOlderThan(timestamp uint64) error
	MessageCacheHits(ids []string) (map[string]bool, error)

	InsertPendingConfirmation(confirmation *RawMessageConfirmation) error
	SaveHashRatchetMessage(groupID []byte, keyID []byte, m *ReceivedMessage) error
	GetHashRatchetMessages(keyID []byte) ([]*ReceivedMessage, error)
	DeleteHashRatchetMessages(ids [][]byte) error
	DeleteHashRatchetMessagesOlderThan(timestamp int64) error
}

type WakuPersistence interface {
	Keys() (map[string][]byte, error)
	AddKey(chatID string, key []byte) error
	InsertProtectedTopic(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error
	DeleteProtectedTopic(pubsubTopic string) error
	FetchPrivateKeyForProtectedTopic(topic string) (*ecdsa.PrivateKey, error)
	ProtectedTopics() ([]ProtectedTopicRecord, error)
}

type SegmentationPersistence interface {
	IsMessageAlreadyCompleted(hash []byte) (bool, error)
	SaveMessageSegment(segment *SegmentMessage, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*SegmentMessage, error)
	CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	RemoveMessageSegmentsOlderThan(timestamp int64) error
	RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error
}

type EncryptionPersistence interface {
	X3DHBundlesStorage() X3DHBundlesPersistence
	DRKeysStorage() DRKeysPersistence
	DRSessionStorage() DRSessionPersistence
	SharedSecretStorage() SharedSecretPersistence
	MultideviceStorage() MultidevicePersistence
	HashRatchetStorage() HashRatchetPersistence
}

type ProtectedTopicRecord struct {
	PubKey *ecdsa.PublicKey
	Topic  string
}

type SignedPreKeyRecord struct {
	SignedPreKey    []byte
	Version         uint32
	ProtocolVersion uint32
}

type BundleRecord struct {
	Identity      []byte
	SignedPreKeys map[string]*SignedPreKeyRecord
	Signature     []byte
	Timestamp     int64
}

type BundleContainerRecord struct {
	Bundle              *BundleRecord
	PrivateSignedPreKey []byte
}

type RatchetInfoRecord struct {
	ID             []byte
	Sk             []byte
	PrivateKey     []byte
	PublicKey      []byte
	Identity       []byte
	BundleID       []byte
	EphemeralKey   []byte
	InstallationID string
}

type X3DHBundlesPersistence interface {
	AddPrivateBundle(bc *BundleContainerRecord) error
	AddPublicBundle(b *BundleRecord) error
	GetAnyPrivateBundle(myIdentityKey []byte, installations []*Installation) (*BundleContainerRecord, error)
	GetPrivateKeyBundle(bundleID []byte) ([]byte, error)
	MarkBundleExpired(identity []byte) error
	GetPublicBundle(publicKey *ecdsa.PublicKey, installations []*Installation) (*BundleRecord, error)
	AddRatchetInfo(key []byte, identity []byte, bundleID []byte, ephemeralKey []byte, installationID string) error
	GetRatchetInfo(bundleID []byte, theirIdentity []byte, installationID string) (*RatchetInfoRecord, error)
	GetAnyRatchetInfo(identity []byte, installationID string) (*RatchetInfoRecord, error)
	RatchetInfoConfirmed(bundleID []byte, theirIdentity []byte, installationID string) error
}

type DRKey []byte

type DRKeysPersistence interface {
	Get(k DRKey, msgNum uint) (mk DRKey, ok bool, err error)
	Put(sessionID []byte, k DRKey, msgNum uint, mk DRKey, keySeqNum uint) error
	DeleteMk(k DRKey, msgNum uint) error
	DeleteOldMks(sessionID []byte, deleteUntilSeqKey uint) error
	TruncateMks(sessionID []byte, maxKeys int) error
	Count(k DRKey) (uint, error)
	All() (map[string]map[uint]DRKey, error)
}

type DRSessionState struct {
	DHr          []byte
	DHsPublic    []byte
	DHsPrivate   []byte
	RootChainKey []byte
	SendChainKey []byte
	SendChainN   uint
	RecvChainKey []byte
	RecvChainN   uint
	Pn           uint
	Step         uint
	KeysCount    uint
}

type DRSessionPersistence interface {
	Save(id []byte, state *DRSessionState) error
	Load(id []byte) (*DRSessionState, error)
}

type SharedSecretResponse struct {
	Secret          []byte
	InstallationIDs map[string]bool
}

type SharedSecretPersistence interface {
	Add(identity []byte, secret []byte, installationID string) error
	Get(identity []byte, installationIDs []string) (*SharedSecretResponse, error)
	All() ([][][]byte, error)
}

type MultidevicePersistence interface {
	GetActiveInstallations(maxInstallations int, identity []byte) ([]*Installation, error)
	GetInstallations(identity []byte) ([]*Installation, error)
	AddInstallations(identity []byte, timestamp int64, installations []*Installation, defaultEnabled bool) ([]*Installation, error)
	EnableInstallation(identity []byte, installationID string) error
	DisableInstallation(identity []byte, installationID string) error
	SetInstallationMetadata(identity []byte, installationID string, metadata *InstallationMetadata) error
	SetInstallationName(identity []byte, installationID string, name string) error
}

type HashRatchetKeyRecord struct {
	GroupID         []byte
	KeyID           []byte
	KeyTimestamp    uint64
	DeprecatedKeyID uint32
	Key             []byte
}

type HRCacheRecord struct {
	GroupID         []byte
	KeyID           []byte
	DeprecatedKeyID uint32
	Key             []byte
	Hash            []byte
	SeqNo           uint32
}

type HashRatchetPersistence interface {
	GetHashRatchetCache(ratchet *HashRatchetKeyRecord, seqNo uint32) (*HRCacheRecord, error)
	GetCurrentKeyForGroup(groupID []byte) (*HashRatchetKeyRecord, error)
	GetKeysForGroup(groupID []byte) ([]*HashRatchetKeyRecord, error)
	SaveHashRatchetKeyHash(ratchet *HashRatchetKeyRecord, hash []byte, seqNo uint32) error
	SaveHashRatchetKey(ratchet *HashRatchetKeyRecord) error
	GetHashRatchetKeyByID(keyID []byte) (*HashRatchetKeyRecord, error)
}
