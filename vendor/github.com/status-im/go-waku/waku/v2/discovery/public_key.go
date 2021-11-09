package discovery

import (
	"crypto/ecdsa"
	"crypto/subtle"
	"encoding/asn1"
	"errors"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/minio/sha256-simd"
)

// Taken from: https://github.com/libp2p/go-libp2p-core/blob/094b0d3f8ba2934339cb35e1a875b11ab6d08839/crypto/ecdsa.go as
// they don't provide a way to set the key
var ErrNilSig = errors.New("sig is nil")

// ECDSASig holds the r and s values of an ECDSA signature
type ECDSASig struct {
	R, S *big.Int
}

// ECDSAPublicKey is an implementation of an ECDSA public key
type ECDSAPublicKey struct {
	pub *ecdsa.PublicKey
}

// Type returns the key type
func (ePub *ECDSAPublicKey) Type() pb.KeyType {
	return pb.KeyType_Secp256k1
}

// Raw returns x509 bytes from a public key
func (ePub *ECDSAPublicKey) Raw() ([]byte, error) {
	return ethcrypto.CompressPubkey(ePub.pub), nil
}

// Bytes returns the public key as protobuf bytes
func (ePub *ECDSAPublicKey) Bytes() ([]byte, error) {
	return crypto.MarshalPublicKey(ePub)
}

// Equals compares to public keys
func (ePub *ECDSAPublicKey) Equals(o crypto.Key) bool {
	return basicEquals(ePub, o)
}

// Verify compares data to a signature
func (ePub *ECDSAPublicKey) Verify(data, sigBytes []byte) (bool, error) {
	sig := new(ECDSASig)
	if _, err := asn1.Unmarshal(sigBytes, sig); err != nil {
		return false, err
	}
	if sig == nil {
		return false, ErrNilSig
	}

	hash := sha256.Sum256(data)

	return ecdsa.Verify(ePub.pub, hash[:], sig.R, sig.S), nil
}

func basicEquals(k1, k2 crypto.Key) bool {
	if k1.Type() != k2.Type() {
		return false
	}

	a, err := k1.Raw()
	if err != nil {
		return false
	}
	b, err := k2.Raw()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
