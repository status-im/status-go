package chat

import (
	"crypto/ecdsa"
)

type RatchetInfo struct {
	// Has the initial message been confirmed?
	InitialX3DHMessage []byte
}

type PersistenceServiceInterface interface {
	GetPublicBundle(*ecdsa.PublicKey) (*Bundle, error)
	AddPublicBundle(*Bundle) error

	GetAnyPrivateBundle() (*Bundle, error)
	GetPrivateBundle([]byte) (*BundleContainer, error)
	AddPrivateBundle(*BundleContainer) error

	GetAnySymmetricKey(*ecdsa.PublicKey) ([]byte, *ecdsa.PublicKey, error)
	GetSymmetricKey(*ecdsa.PublicKey, *ecdsa.PublicKey) ([]byte, error)
	AddSymmetricKey(*ecdsa.PublicKey, *ecdsa.PublicKey, []byte) error

	GetRatchetInfo(*ecdsa.PublicKey) (*RatchetInfo, error)
}
