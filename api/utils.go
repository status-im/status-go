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

	bls12381 "github.com/kilic/bls12-381"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-varint"

	"github.com/status-im/status-go/eth-node/crypto"
)

const (
	secp256k1KeyType   = 0xe7
	bls12p381g1KeyType = 0xea
	bls12p381g2KeyType = 0xeb
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

	out, err := encode(base, cpk)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DecompressPublicKey
func DecompressPublicKey(key string) ([]byte, error) {
	cpk, err := decode(key)
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
	switch keyType {
	case secp256k1KeyType:
		return compressSecp256k1PublicKey(key)

	case bls12p381g1KeyType:
		return compressBls12p381g1PublicKey(key)

	case bls12p381g2KeyType:
		return compressBls12p381g2PublicKey(key)

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

func compressBls12p381g1PublicKey(key []byte) ([]byte, error) {
	g1 := bls12381.NewG1()

	// Generate the G1 point
	pg1, err := g1.FromBytes(key)
	if err != nil {
		return nil, err
	}

	cpk := g1.ToCompressed(pg1)
	return cpk, nil
}

func compressBls12p381g2PublicKey(key []byte) ([]byte, error) {
	g2 := bls12381.NewG2()

	// Generate the G2 point
	pg2, err := g2.FromBytes(key)
	if err != nil {
		return nil, err
	}

	cpk := g2.ToCompressed(pg2)
	return cpk, nil
}

func decompressPublicKey(key []byte, keyType uint64) ([]byte, error) {
	switch keyType {
	case secp256k1KeyType:
		return decompressSecp256k1PublicKey(key)

	case bls12p381g1KeyType:
		return decompressBls12p381g1PublicKey(key)

	case bls12p381g2KeyType:
		return decompressBls12p381g2PublicKey(key)

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

func decompressBls12p381g1PublicKey(key []byte) ([]byte, error){
	g1 := bls12381.NewG1()
	pg1, err := g1.FromCompressed(key)
	if err != nil {
		return nil, err
	}

	pk := g1.ToUncompressed(pg1)
	return pk, nil
}

func decompressBls12p381g2PublicKey(key []byte) ([]byte, error){
	g2 := bls12381.NewG2()
	pg2, err := g2.FromCompressed(key)
	if err != nil {
		return nil, err
	}

	pk := g2.ToUncompressed(pg2)
	return pk, nil
}

func encode(base string, data []byte) (string, error) {
	if base == "0x" {
		base = "f"
	}
	return multibase.Encode(multibase.Encoding(base[0]), data)
}

func decode(data string) ([]byte, error) {
	if data[0:2] == "0x" {
		data = "f" + data[2:]
	}

	_, dd, err := multibase.Decode(data)
	return dd, err
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
