package types

import "crypto/ecdsa"

type SharedSecret struct {
	Identity *ecdsa.PublicKey
	Key      []byte
}
