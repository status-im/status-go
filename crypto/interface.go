package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/status-im/status-go/crypto/types"
)

const SignatureLength = 64 + 1 // 64 bytes ECDSA signature + 1 byte recovery id

// CryptoProvider defines the interface for cryptographic operations
type CryptoProvider interface {
	// S256 returns an instance of the secp256k1 curve.
	S256() elliptic.Curve

	// GenerateKey generates a new ECDSA private key.
	GenerateKey() (*ecdsa.PrivateKey, error)

	// HexToECDSA converts a hex string to an ECDSA private key
	HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error)

	// FromECDSA converts an ECDSA private key to bytes
	FromECDSA(prv *ecdsa.PrivateKey) []byte

	// FromECDSAPub converts an ECDSA public key to bytes
	FromECDSAPub(pub *ecdsa.PublicKey) []byte

	// PubkeyToAddress derives an Ethereum address from a public key
	PubkeyToAddress(p ecdsa.PublicKey) types.Address

	// Keccak256 computes the Keccak256 hash of the concatenated data
	Keccak256(data ...[]byte) []byte

	// ToECDSAUnsafe converts bytes to an ECDSA private key (unsafe - no validation)
	ToECDSAUnsafe(data []byte) *ecdsa.PrivateKey

	// ToECDSA creates a private key with the given D value.
	ToECDSA(d []byte) (*ecdsa.PrivateKey, error)

	// TextHash is a helper function that calculates a hash for the given message that can be safely used to calculate a signature from.
	//
	// The hash is calulcated as
	//
	//	keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
	//
	// This gives context to the signed message and prevents signing of transactions.
	TextHash(data []byte) []byte

	// TextAndHash is a helper function that calculates a hash for the given message that can be safely used to calculate a signature from.
	//
	// The hash is calulcated as
	//
	//	keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
	//
	// This gives context to the signed message and prevents signing of transactions.
	TextAndHash(data []byte) ([]byte, string)

	// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
	// converting it to an internal Hash data structure.
	Keccak256Hash(data ...[]byte) (h types.Hash)

	// Sign calculates an ECDSA signature.
	//
	// This function is susceptible to chosen plaintext attacks that can leak
	// information about the private key that is used for signing. Callers must
	// be aware that the given digest cannot be chosen by an adversery. Common
	// solution is to hash any input before calculating the signature.
	//
	// The produced signature is in the [R || S || V] format where V is 0 or 1.
	Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error)

	// SigToPub returns the public key that created the given signature.
	SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error)

	// UnmarshalPubkey converts bytes to a secp256k1 public key.
	UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error)

	// CreateAddress creates an ethereum address given the bytes and the nonce
	CreateAddress(b types.Address, nonce uint64) types.Address

	// DecompressPubkey decompresses a public key from the 33-byte compressed format.
	DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error)

	// CompressPubkey encodes a public key to the 33-byte compressed format.
	CompressPubkey(pubkey *ecdsa.PublicKey) []byte

	// GenerateSharedKey generates a shared key given a private and a public key
	GenerateSharedKey(myIdentityKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey, sskLen int) ([]byte, error)
}
