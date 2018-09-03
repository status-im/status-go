package chat

import (
	"crypto/ecdsa"

	dr "github.com/status-im/doubleratchet"
)

// RatchetInfo holds the current ratchet state
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

// PersistenceServiceInterface defines the interface for a storage service
type PersistenceServiceInterface interface {
	// GetKeysStorage returns the associated double ratchet KeysStorage object
	GetKeysStorage() dr.KeysStorage
	// GetSessionStorage returns the associated double ratchet SessionStorage object
	GetSessionStorage() dr.SessionStorage

	// GetPublicBundle retrieves an existing Bundle for the specified public key
	GetPublicBundle(*ecdsa.PublicKey) (*Bundle, error)
	// AddPublicBundle persists a specified Bundle
	AddPublicBundle(*Bundle) error

	// GetAnyPrivateBundle retrieves any bundle containing a private key
	GetAnyPrivateBundle([]byte) (*BundleContainer, error)
	// GetPrivateKeyBundle retrieves a BundleContainer with the specified signed prekey
	GetPrivateKeyBundle([]byte) ([]byte, error)
	// AddPrivateBundle persists a BundleContainer
	AddPrivateBundle(*BundleContainer) error
	// MarkBundleExpired marks a bundle as expired, not to be used for encryption anymore
	MarkBundleExpired([]byte) error

	// AddRatchetInfo persists the specified ratchet info
	AddRatchetInfo([]byte, []byte, []byte, []byte, string) error
	// GetRatchetInfo retrieves the existing RatchetInfo for a specified bundle ID and interlocutor public key
	GetRatchetInfo([]byte, []byte) (*RatchetInfo, error)
	// GetAnyRatchetInfo retrieves any existing RatchetInfo for a specified interlocutor public key
	GetAnyRatchetInfo([]byte, string) (*RatchetInfo, error)
	// RatchetInfoConfirmed clears the ephemeral key in the RatchetInfo
	// associated with the specified bundle ID and interlocutor identity public key
	RatchetInfoConfirmed([]byte, []byte) error
}
