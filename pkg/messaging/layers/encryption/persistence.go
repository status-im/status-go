package encryption

import (
	"crypto/ecdsa"

	dr "github.com/status-im/doubleratchet"

	multidevice2 "github.com/status-im/status-go/pkg/messaging/layers/encryption/multidevice"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption/sharedsecret"
)

// RatchetInfo holds the current ratchet state.
type RatchetInfo struct {
	ID             []byte
	Sk             []byte
	PrivateKey     []byte
	PublicKey      []byte
	Identity       []byte
	BundleID       []byte
	EphemeralKey   []byte
	InstallationID string
}

type HRCache struct {
	GroupID         []byte
	KeyID           []byte
	DeprecatedKeyID uint32
	Key             []byte
	Hash            []byte
	SeqNo           uint32
}

type Persistence interface {
	KeysStorage() dr.KeysStorage
	SessionStorage() dr.SessionStorage
	SharedSecretStorage() sharedsecret.Persistence
	MultideviceStorage() multidevice2.Persistence

	AddPrivateBundle(bc *BundleContainer) error
	AddPublicBundle(b *Bundle) error
	GetAnyPrivateBundle(myIdentityKey []byte, installations []*multidevice2.Installation) (*BundleContainer, error)
	GetPrivateKeyBundle(bundleID []byte) ([]byte, error)
	MarkBundleExpired(identity []byte) error
	GetPublicBundle(publicKey *ecdsa.PublicKey, installations []*multidevice2.Installation) (*Bundle, error)
	AddRatchetInfo(key []byte, identity []byte, bundleID []byte, ephemeralKey []byte, installationID string) error
	GetRatchetInfo(bundleID []byte, theirIdentity []byte, installationID string) (*RatchetInfo, error)
	GetAnyRatchetInfo(identity []byte, installationID string) (*RatchetInfo, error)
	RatchetInfoConfirmed(bundleID []byte, theirIdentity []byte, installationID string) error
	GetHashRatchetCache(ratchet *HashRatchetKeyCompatibility, seqNo uint32) (*HRCache, error)
	GetCurrentKeyForGroup(groupID []byte) (*HashRatchetKeyCompatibility, error)
	GetKeysForGroup(groupID []byte) ([]*HashRatchetKeyCompatibility, error)
	SaveHashRatchetKeyHash(ratchet *HashRatchetKeyCompatibility, hash []byte, seqNo uint32) error
	SaveHashRatchetKey(ratchet *HashRatchetKeyCompatibility) error
	GetHashRatchetKeyByID(keyID []byte) (*HashRatchetKeyCompatibility, error)
}
