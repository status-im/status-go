package multiformat

import (
	"crypto/elliptic"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"

	bls12381 "github.com/kilic/bls12-381"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-varint"
)

const (
	secp256k1KeyType   = 0xe7
	bls12p381g1KeyType = 0xea
	bls12p381g2KeyType = 0xeb
)

// CompressPublicKey compresses an uncompressed multibase encoded multicodec identified EC public key
// For details on usage see specs //TODO add the link to the specs
func CompressPublicKey(key, outputBase string) (string, error) {
	dKey, err := multibaseDecode(key)
	if err != nil {
		return "", err
	}

	kt, i, err := getPublicKeyType(dKey)
	if err != nil {
		return "", err
	}

	cpk, err := compressPublicKey(dKey[i:], kt)
	if err != nil {
		return "", err
	}

	cpk = prependKeyIdentifier(cpk, kt, i)

	return multibaseEncode(outputBase, cpk)
}

// DecompressPublicKey decompresses a compressed multibase encoded multicodec identified EC public key
// For details on usage see specs //TODO add the link to the specs
func DecompressPublicKey(key, outputBase string) (string, error) {
	cpk, err := multibaseDecode(key)
	if err != nil {
		return "", err
	}

	kt, i, err := getPublicKeyType(cpk)
	if err != nil {
		return "", err
	}

	pk, err := decompressPublicKey(cpk[i:], kt)
	if err != nil {
		return "", err
	}

	pk = prependKeyIdentifier(pk, kt, i)

	return multibaseEncode(outputBase, pk)
}

// getPublicKeyType wrapper for the `varint.FromUvarint()` func
func getPublicKeyType(key []byte) (uint64, int, error) {
	return varint.FromUvarint(key)
}

// prependKeyIdentifier prepends an Unsigned Variable Integer (uvarint) to a given []byte
func prependKeyIdentifier(key []byte, kt uint64, ktl int) []byte {
	buf := make([]byte, ktl)
	varint.PutUvarint(buf, kt)

	key = append(buf, key...)
	return key
}

// compressPublicKey serves as logic switch function to parse key data for compression based on the given keyType
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

// compressSecp256k1PublicKey is a dedicated key compression function for secp256k1 pks
func compressSecp256k1PublicKey(key []byte) ([]byte, error) {
	x, y := elliptic.Unmarshal(secp256k1.S256(), key)

	if err := isSecp256k1XYValid(key, x, y); err != nil {
		return nil, err
	}

	cpk := secp256k1.CompressPubkey(x, y)

	return cpk, nil
}

// compressBls12p381g1PublicKey is a dedicated key compression function for bls12 381 g1 pks
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

// compressBls12p381g1PublicKey is a dedicated key compression function for bls12 381 g2 pks
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

// decompressPublicKey serves as logic switch function to parse key data for decompression based on the given keyType
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

// decompressSecp256k1PublicKey is a dedicated key decompression function for secp256k1 pks
func decompressSecp256k1PublicKey(key []byte) ([]byte, error) {
	x, y := secp256k1.DecompressPubkey(key)

	if err := isSecp256k1XYValid(key, x, y); err != nil {
		return nil, err
	}

	k := elliptic.Marshal(secp256k1.S256(), x, y)

	return k, nil
}

// isSecp256k1XYValid checks if a given x and y coordinate is nil, returns an error if either x or y is nil
// secp256k1.DecompressPubkey will not return an error if a compressed pk fails decompression and instead returns
// nil x, y coordinates
func isSecp256k1XYValid(key []byte, x, y *big.Int) error {
	if x == nil || y == nil {
		return fmt.Errorf("invalid public key format, '%b'", key)
	}

	return nil
}

// decompressBls12p381g1PublicKey is a dedicated key decompression function for bls12 381 g1 pks
func decompressBls12p381g1PublicKey(key []byte) ([]byte, error) {
	g1 := bls12381.NewG1()
	pg1, err := g1.FromCompressed(key)
	if err != nil {
		return nil, err
	}

	pk := g1.ToUncompressed(pg1)
	return pk, nil
}

// decompressBls12p381g2PublicKey is a dedicated key decompression function for bls12 381 g2 pks
func decompressBls12p381g2PublicKey(key []byte) ([]byte, error) {
	g2 := bls12381.NewG2()
	pg2, err := g2.FromCompressed(key)
	if err != nil {
		return nil, err
	}

	pk := g2.ToUncompressed(pg2)
	return pk, nil
}

// multibaseEncode wraps `multibase.Encode()` extending the base functionality to support `0x` prefixed strings
func multibaseEncode(base string, data []byte) (string, error) {
	if base == "0x" {
		base = "f"
	}
	return multibase.Encode(multibase.Encoding(base[0]), data)
}

// multibaseDecode wraps `multibase.Decode()` extending the base functionality to support `0x` prefixed strings
func multibaseDecode(data string) ([]byte, error) {
	if data[0:2] == "0x" {
		data = "f" + data[2:]
	}

	_, dd, err := multibase.Decode(data)
	return dd, err
}
