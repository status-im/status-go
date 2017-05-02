package notifications

import (
	"crypto/sha512"
	"errors"

	crand "crypto/rand"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"golang.org/x/crypto/pbkdf2"
)

// makeSessionKey returns pseudo-random symmetric key, which is used as
// session key between notification client and server
func makeSessionKey() ([]byte, error) {
	// generate random key
	const keyLen = 32
	const size = keyLen * 2
	buf := make([]byte, size)
	_, err := crand.Read(buf)
	if err != nil {
		return nil, err
	} else if !validateSymmetricKey(buf) {
		return nil, errors.New("error in GenerateSymKey: crypto/rand failed to generate random data")
	}

	key := buf[:keyLen]
	salt := buf[keyLen:]
	derived, err := whisper.DeriveOneTimeKey(key, salt, whisper.EnvelopeVersion)
	if err != nil {
		return nil, err
	} else if !validateSymmetricKey(derived) {
		return nil, errors.New("failed to derive valid key")
	}

	return derived, nil
}

// validateSymmetricKey returns false if the key contains all zeros
func validateSymmetricKey(k []byte) bool {
	return len(k) > 0 && !containsOnlyZeros(k)
}

// containsOnlyZeros checks if data is empty or not
func containsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// MakeTopic returns Whisper topic *as bytes array* by generating cryptographic key from the provided password
func MakeTopicAsBytes(password []byte) ([]byte) {
	topic := make([]byte, int(whisper.TopicLength))
	x := pbkdf2.Key(password, password, 8196, 128, sha512.New)
	for i := 0; i < len(x); i++ {
		topic[i%whisper.TopicLength] ^= x[i]
	}

	return topic
}

// MakeTopic returns Whisper topic by generating cryptographic key from the provided password
func MakeTopic(password []byte) (topic whisper.TopicType) {
	x := pbkdf2.Key(password, password, 8196, 128, sha512.New)
	for i := 0; i < len(x); i++ {
		topic[i%whisper.TopicLength] ^= x[i]
	}

	return
}
