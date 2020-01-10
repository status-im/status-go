// +build openssl

package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"

	btcec "github.com/btcsuite/btcd/btcec"
	openssl "github.com/libp2p/go-openssl"
	"golang.org/x/crypto/ed25519"
)

// KeyPairFromStdKey wraps standard library (and secp256k1) private keys in libp2p/go-libp2p-core/crypto keys
func KeyPairFromStdKey(priv crypto.PrivateKey) (PrivKey, PubKey, error) {
	if priv == nil {
		return nil, nil, ErrNilPrivateKey
	}

	switch p := priv.(type) {
	case *rsa.PrivateKey:
		pk, err := openssl.LoadPrivateKeyFromDER(x509.MarshalPKCS1PrivateKey(p))
		if err != nil {
			return nil, nil, err
		}

		return &opensslPrivateKey{pk}, &opensslPublicKey{pk}, nil

	case *ecdsa.PrivateKey:
		return &ECDSAPrivateKey{p}, &ECDSAPublicKey{&p.PublicKey}, nil

	case *ed25519.PrivateKey:
		pubIfc := p.Public()
		pub, _ := pubIfc.(ed25519.PublicKey)
		return &Ed25519PrivateKey{*p}, &Ed25519PublicKey{pub}, nil

	case *btcec.PrivateKey:
		sPriv := Secp256k1PrivateKey(*p)
		sPub := Secp256k1PublicKey(*p.PubKey())
		return &sPriv, &sPub, nil

	default:
		return nil, nil, ErrBadKeyType
	}
}
