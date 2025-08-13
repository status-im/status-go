package geth

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/status-im/status-go/crypto/types"
)

// GethCryptoProvider implements the CryptoProvider interface using go-ethereum
type GethCryptoProvider struct{}

// NewGethCryptoProvider creates a new GethCryptoProvider instance
func NewGethCryptoProvider() *GethCryptoProvider {
	return &GethCryptoProvider{}
}

func (g *GethCryptoProvider) S256() elliptic.Curve {
	return secp256k1.S256()
}

func (g *GethCryptoProvider) GenerateKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

func (g *GethCryptoProvider) HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(hexkey)
}

func (g *GethCryptoProvider) ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return crypto.ToECDSA(d)
}

func (g *GethCryptoProvider) FromECDSA(prv *ecdsa.PrivateKey) []byte {
	return crypto.FromECDSA(prv)
}

func (g *GethCryptoProvider) FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	return crypto.FromECDSAPub(pub)
}

func (g *GethCryptoProvider) PubkeyToAddress(p ecdsa.PublicKey) types.Address {
	address := crypto.PubkeyToAddress(p)
	return types.Address(address)
}

func (g *GethCryptoProvider) Keccak256(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}

func (g *GethCryptoProvider) ToECDSAUnsafe(data []byte) *ecdsa.PrivateKey {
	return crypto.ToECDSAUnsafe(data)
}

func (g *GethCryptoProvider) TextHash(data []byte) []byte {
	return accounts.TextHash(data)
}

func (g *GethCryptoProvider) TextAndHash(data []byte) ([]byte, string) {
	return accounts.TextAndHash(data)
}

func (g *GethCryptoProvider) Keccak256Hash(data ...[]byte) (h types.Hash) {
	return types.Hash(crypto.Keccak256Hash(data...))
}

func (g *GethCryptoProvider) Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	return crypto.Sign(digestHash, prv)
}

func (g *GethCryptoProvider) SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	return crypto.SigToPub(hash, sig)
}

func (g *GethCryptoProvider) UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	return crypto.UnmarshalPubkey(pub)
}

func (g *GethCryptoProvider) CreateAddress(b types.Address, nonce uint64) types.Address {
	return types.Address(crypto.CreateAddress(common.Address(b), nonce))
}

func (g *GethCryptoProvider) DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	return crypto.DecompressPubkey(pubkey)
}

func (g *GethCryptoProvider) CompressPubkey(pubkey *ecdsa.PublicKey) []byte {
	return crypto.CompressPubkey(pubkey)
}

// GenerateSharedKey generates a shared key given a private and a public key
func (g *GethCryptoProvider) GenerateSharedKey(myIdentityKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey, sskLen int) ([]byte, error) {
	eciesPrivate := ecies.ImportECDSA(myIdentityKey)
	eciesPublic := ecies.ImportECDSAPublic(theirEphemeralKey)

	return eciesPrivate.GenerateShared(
		eciesPublic,
		sskLen,
		sskLen,
	)
}
