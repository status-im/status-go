package common

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// IsPubKeyEqual checks that two public keys are equal
func IsPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	if !ValidatePublicKey(a) {
		return false
	} else if !ValidatePublicKey(b) {
		return false
	}
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}

// ValidatePublicKey checks the format of the given public key.
func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

// BytesToUintLittleEndian converts the slice to 64-bit unsigned integer.
func BytesToUintLittleEndian(b []byte) (res uint64) {
	mul := uint64(1)
	for i := 0; i < len(b); i++ {
		res += uint64(b[i]) * mul
		mul *= 256
	}
	return res
}

// BytesToUintBigEndian converts the slice to 64-bit unsigned integer.
func BytesToUintBigEndian(b []byte) (res uint64) {
	for i := 0; i < len(b); i++ {
		res *= 256
		res += uint64(b[i])
	}
	return res
}

// ContainsOnlyZeros checks if the data contain only zeros.
func ContainsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// GenerateRandomID generates a random string, which is then returned to be used as a key id
func GenerateRandomID() (id string, err error) {
	buf, err := GenerateSecureRandomData(KeyIDSize)
	if err != nil {
		return "", err
	}
	if !ValidateDataIntegrity(buf, KeyIDSize) {
		return "", fmt.Errorf("error in generateRandomID: crypto/rand failed to generate random data")
	}
	id = common.Bytes2Hex(buf)
	return id, err
}

// ValidateDataIntegrity returns false if the data have the wrong or contains all zeros,
// which is the simplest and the most common bug.
func ValidateDataIntegrity(k []byte, expectedSize int) bool {
	if len(k) != expectedSize {
		return false
	}
	if expectedSize > 3 && ContainsOnlyZeros(k) {
		return false
	}
	return true
}
