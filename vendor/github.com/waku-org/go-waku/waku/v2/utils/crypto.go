package utils

import (
	"crypto/ecdsa"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/libp2p/go-libp2p/core/crypto"
)

// EcdsaPubKeyToSecp256k1PublicKey converts an `ecdsa.PublicKey` into a libp2p `crypto.Secp256k1PublicKey“
func EcdsaPubKeyToSecp256k1PublicKey(pubKey *ecdsa.PublicKey) *crypto.Secp256k1PublicKey {
	xFieldVal := &secp256k1.FieldVal{}
	yFieldVal := &secp256k1.FieldVal{}
	xFieldVal.SetByteSlice(pubKey.X.Bytes())
	yFieldVal.SetByteSlice(pubKey.Y.Bytes())
	return (*crypto.Secp256k1PublicKey)(secp256k1.NewPublicKey(xFieldVal, yFieldVal))
}

// EcdsaPrivKeyToSecp256k1PrivKey converts an `ecdsa.PrivateKey` into a libp2p `crypto.Secp256k1PrivateKey“
func EcdsaPrivKeyToSecp256k1PrivKey(privKey *ecdsa.PrivateKey) *crypto.Secp256k1PrivateKey {
	privK := secp256k1.PrivKeyFromBytes(privKey.D.Bytes())
	return (*crypto.Secp256k1PrivateKey)(privK)
}
