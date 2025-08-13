package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/status-im/status-go/crypto/geth"
	"github.com/status-im/status-go/crypto/types"
)

var defaultProvider CryptoProvider = geth.NewGethCryptoProvider()

const (
	// Shared secret key length
	sskLen = 16

	aesNonceLength = 12
)

// SetProvider sets the default crypto provider
func SetProvider(provider CryptoProvider) {
	defaultProvider = provider
}

func S256() elliptic.Curve {
	return defaultProvider.S256()
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return defaultProvider.GenerateKey()
}

func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	return defaultProvider.HexToECDSA(hexkey)
}

func FromECDSA(prv *ecdsa.PrivateKey) []byte {
	return defaultProvider.FromECDSA(prv)
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	return defaultProvider.FromECDSAPub(pub)
}

func PubkeyToAddress(p ecdsa.PublicKey) types.Address {
	return defaultProvider.PubkeyToAddress(p)
}

func Keccak256(data ...[]byte) []byte {
	return defaultProvider.Keccak256(data...)
}

func ToECDSAUnsafe(data []byte) *ecdsa.PrivateKey {
	return defaultProvider.ToECDSAUnsafe(data)
}

func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return defaultProvider.ToECDSA(d)
}

func TextHash(data []byte) []byte {
	return defaultProvider.TextHash(data)
}

func TextAndHash(data []byte) ([]byte, string) {
	return defaultProvider.TextAndHash(data)
}

func Keccak256Hash(data ...[]byte) (h types.Hash) {
	return defaultProvider.Keccak256Hash(data...)
}

func Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	return defaultProvider.Sign(digestHash, prv)
}

func SignBytes(data []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	return Sign(Keccak256(data), prv)
}

func SignBytesAsHex(data []byte, identity *ecdsa.PrivateKey) (string, error) {
	signature, err := SignBytes(data, identity)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(signature), nil
}

func SignStringAsHex(data string, identity *ecdsa.PrivateKey) (string, error) {
	return SignBytesAsHex([]byte(data), identity)
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	return defaultProvider.SigToPub(hash, sig)
}

func CreateAddress(b types.Address, nonce uint64) types.Address {
	return defaultProvider.CreateAddress(b, nonce)
}

func DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	return defaultProvider.DecompressPubkey(pubkey)
}

func CompressPubkey(pubkey *ecdsa.PublicKey) []byte {
	return defaultProvider.CompressPubkey(pubkey)
}

func UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	return defaultProvider.UnmarshalPubkey(pub)
}

func VerifySignatures(signaturePairs [][3]string) error {
	for _, signaturePair := range signaturePairs {
		content := Keccak256([]byte(signaturePair[0]))

		signature, err := hex.DecodeString(signaturePair[1])
		if err != nil {
			return err
		}

		publicKeyBytes, err := hex.DecodeString(signaturePair[2])
		if err != nil {
			return err
		}

		publicKey, err := UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return err
		}

		recoveredKey, err := SigToPub(
			content,
			signature,
		)
		if err != nil {
			return err
		}

		if PubkeyToAddress(*recoveredKey) != PubkeyToAddress(*publicKey) {
			return errors.New("identity key and signature mismatch")
		}
	}

	return nil
}

func ExtractSignatures(signaturePairs [][2]string) ([]string, error) {
	response := make([]string, len(signaturePairs))
	for i, signaturePair := range signaturePairs {
		content := Keccak256([]byte(signaturePair[0]))

		signature, err := hex.DecodeString(signaturePair[1])
		if err != nil {
			return nil, err
		}

		recoveredKey, err := SigToPub(
			content,
			signature,
		)
		if err != nil {
			return nil, err
		}

		response[i] = fmt.Sprintf("%x", FromECDSAPub(recoveredKey))
	}

	return response, nil
}

// ExtractSignature returns a public key for a given data and signature.
func ExtractSignature(data, signature []byte) (*ecdsa.PublicKey, error) {
	dataHash := Keccak256(data)
	return SigToPub(dataHash, signature)
}

func EcRecover(data types.HexBytes, sig types.HexBytes) (types.Address, error) {
	// Returns the address for the Account that was used to create the signature.
	//
	// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
	// the address of:
	// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")
	// addr = ecrecover(hash, signature)
	//
	// Note, the signature must conform to the secp256k1 curve R, S and V values, where
	// the V value must be be 27 or 28 for legacy reasons.
	//
	// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
	if len(sig) != 65 {
		return types.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}
	if sig[64] != 27 && sig[64] != 28 {
		return types.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1
	hash := TextHash(data)
	rpk, err := SigToPub(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	return PubkeyToAddress(*rpk), nil
}

func GenerateSharedKey(myIdentityKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey) ([]byte, error) {
	return defaultProvider.GenerateSharedKey(myIdentityKey, theirEphemeralKey, sskLen)
}

// DecryptWithDH decrypts message sent with a DH key exchange, and throws away the key after decryption
func DecryptWithDH(myIdentityKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	key, err := GenerateSharedKey(myIdentityKey, theirEphemeralKey)
	if err != nil {
		return nil, err
	}

	return DecryptSymmetric(key, payload)
}

func DecryptSymmetric(key []byte, cyphertext []byte) ([]byte, error) {
	// symmetric messages are expected to contain the 12-byte nonce at the end of the payload
	if len(cyphertext) < aesNonceLength {
		return nil, errors.New("missing salt or invalid payload in symmetric message")
	}
	salt := cyphertext[len(cyphertext)-aesNonceLength:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	decrypted, err := aesgcm.Open(nil, salt, cyphertext[:len(cyphertext)-aesNonceLength], nil)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

func EncryptSymmetric(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	salt, err := generateSecureRandomData(aesNonceLength)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	encrypted := aesgcm.Seal(nil, salt, plaintext, nil)
	return append(encrypted, salt...), nil
}

func containsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

func validateDataIntegrity(k []byte, expectedSize int) bool {
	if len(k) != expectedSize {
		return false
	}
	if containsOnlyZeros(k) {
		return false
	}
	return true
}

func generateSecureRandomData(length int) ([]byte, error) {
	res := make([]byte, length)

	_, err := rand.Read(res)
	if err != nil {
		return nil, err
	}

	if !validateDataIntegrity(res, length) {
		return nil, errors.New("crypto/rand failed to generate secure random data")
	}

	return res, nil
}
