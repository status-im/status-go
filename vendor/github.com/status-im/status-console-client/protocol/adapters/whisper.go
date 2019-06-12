package adapters

import "crypto/ecdsa"

type keysManager interface {
	PrivateKey() *ecdsa.PrivateKey
	AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error)
	AddOrGetSymKeyFromPassword(password string) (string, error)
	GetRawSymKey(string) ([]byte, error)
}
