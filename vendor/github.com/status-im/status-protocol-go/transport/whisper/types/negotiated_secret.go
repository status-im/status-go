package whispertypes

import (
	"crypto/ecdsa"
)

type NegotiatedSecret struct {
	PublicKey *ecdsa.PublicKey
	Key       []byte
}
