package alias

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/status-im/status-go/api/multiformat"
)

const poly uint64 = 0xB8

func generate(seed uint64) string {
	generator := newLSFR(poly, seed)
	adjective1Index := generator.next() % uint64(len(adjectives))
	adjective2Index := generator.next() % uint64(len(adjectives))
	animalIndex := generator.next() % uint64(len(animals))
	adjective1 := adjectives[adjective1Index]
	adjective2 := adjectives[adjective2Index]
	animal := animals[animalIndex]

	return fmt.Sprintf("%s %s %s", adjective1, adjective2, animal)
}

// GenerateFromPublicKey returns the 3 words name given an *ecdsa.PublicKey
func GenerateFromPublicKey(publicKey *ecdsa.PublicKey) string {
	// Here we truncate the public key to the least significant 64 bits
	return generate(uint64(publicKey.X.Int64()))
}

// GenerateFromPublicKeyString calculates the compressed key for the given publicKeyString
// and returns its first 8 chars followed by an ellipsis char and its last 5 chars.
func GenerateFromPublicKeyString(publicKeyString string) (string, error) {
	//  Ensure there's `0x` prefix
	if !strings.HasPrefix(publicKeyString, "0x") {
		publicKeyString = "0x" + publicKeyString
	}

	compressedKey, err := multiformat.SerializeLegacyKey(publicKeyString)
	if err != nil {
		return "", err
	}

	return ShortenedCompressedKey(compressedKey), nil
}

func ShortenedCompressedKey(compressedKey string) string {
	if len(compressedKey) <= 12 {
		return ""
	}
	prefix := compressedKey[0:8]
	suffix := compressedKey[len(compressedKey)-5:]
	return prefix + "…" + suffix
}
