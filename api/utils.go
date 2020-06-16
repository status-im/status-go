package api

import (
	"bytes"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-varint"

	"github.com/status-im/status-go/eth-node/crypto"
)

const (
	SECP256K1_KEY    = 0xe7
	BLS12_381_G1_KEY = 0xea
	BLS12_381_G2_KEY = 0xeb
)

// RunAsync runs the specified function asynchronously.
func RunAsync(f func() error) <-chan error {
	resp := make(chan error, 1)
	go func() {
		err := f()
		resp <- err
		close(resp)
	}()
	return resp
}

// HashMessage calculates the hash of a message to be safely signed by the keycard
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
// This gives context to the signed message and prevents signing of transactions.
func HashMessage(message string) ([]byte, error) {
	buf := bytes.NewBufferString("\x19Ethereum Signed Message:\n")
	if value, ok := decodeHexStrict(message); ok {
		if _, err := buf.WriteString(strconv.Itoa(len(value))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(value); err != nil {
			return nil, err
		}
	} else {
		if _, err := buf.WriteString(strconv.Itoa(len(message))); err != nil {
			return nil, err
		}
		if _, err := buf.WriteString(message); err != nil {
			return nil, err
		}
	}

	return crypto.Keccak256(buf.Bytes()), nil
}

// CompressPublicKey
func CompressPublicKey(base string, key []byte) (string, error) {
	kt, i, err := getPublicKeyType(key)
	if err != nil {
		return "", err
	}

	cpk, err := compressPublicKey(key[i:], kt)
	if err != nil {
		return "", err
	}

	cpk = prependKeyIdentifier(cpk, kt, i)

	out, err := multibase.Encode(multibase.Encoding(base[0]), cpk)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DecompressPublicKey
func DecompressPublicKey(key string) ([]byte, error) {
	_, cpk, err := multibase.Decode(key)
	if err != nil {
		return nil, err
	}

	kt, i, err := getPublicKeyType(cpk)
	if err != nil {
		return nil, err
	}

	pk, err := decompressPublicKey(cpk[i:], kt)
	if err != nil {
		return nil, err
	}

	pk = prependKeyIdentifier(pk, kt, i)

	return pk, nil
}

func getPublicKeyType(key []byte) (uint64, int, error) {
	return varint.FromUvarint(key)
}

func prependKeyIdentifier(key []byte, kt uint64, ktl int) []byte {
	buf := make([]byte, ktl)
	varint.PutUvarint(buf, kt)

	key = append(buf, key...)
	return key
}

func compressPublicKey(key []byte, keyType uint64) ([]byte, error) {
	switch keyType{
	case SECP256K1_KEY:
		return compressSecp256k1PublicKey(key)

	case BLS12_381_G1_KEY:
		return nil, fmt.Errorf("bls12 381 g1 public key not supported")

	case BLS12_381_G2_KEY:
		return nil, fmt.Errorf("bls12 381 g2 public key not supported")

	default:
		return nil, fmt.Errorf("unsupported public key type '%X'", keyType)
	}
}

func compressSecp256k1PublicKey(key []byte) ([]byte, error) {
	x, y := elliptic.Unmarshal(secp256k1.S256(), key)

	if err := isSecp256k1XYValid(key, x, y); err != nil {
		return nil, err
	}

	cpk := secp256k1.CompressPubkey(x, y)

	return cpk, nil
}

func decompressPublicKey(key []byte, keyType uint64) ([]byte, error) {
	switch keyType{
	case SECP256K1_KEY:
		return decompressSecp256k1PublicKey(key)

	case BLS12_381_G1_KEY:
		return nil, fmt.Errorf("bls12 381 g1 public key not supported")

	case BLS12_381_G2_KEY:
		return nil, fmt.Errorf("bls12 381 g2 public key not supported")

	default:
		return nil, fmt.Errorf("unsupported public key type '%X'", keyType)
	}
}

func decompressSecp256k1PublicKey(key []byte) ([]byte, error) {
	x, y := secp256k1.DecompressPubkey(key)

	if err := isSecp256k1XYValid(key, x, y); err != nil {
		return nil, err
	}

	k := elliptic.Marshal(secp256k1.S256(), x, y)

	return k, nil
}

func isSecp256k1XYValid(key []byte, x, y *big.Int) error {
	if x == nil || y == nil {
		return fmt.Errorf("invalid public key format, '%b'", key)
	}

	return nil
}

func decodeHexStrict(s string) ([]byte, bool) {
	if !strings.HasPrefix(s, "0x") {
		return nil, false
	}

	value, err := hex.DecodeString(s[2:])
	if err != nil {
		return nil, false
	}

	return value, true
}
