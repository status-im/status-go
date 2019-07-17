package statusproto

import "crypto/ecdsa"

type Chat interface {
	ID() string
	PublicName() string
	PublicKey() *ecdsa.PublicKey
}
