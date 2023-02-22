package pb

import (
	"crypto/sha256"

	proto "google.golang.org/protobuf/proto"
)

// Hash calculates the hash of a waku message
func (msg *WakuMessage) Hash() ([]byte, int, error) {
	out, err := proto.Marshal(msg)
	if err != nil {
		return nil, 0, err
	}

	return Hash(out), len(out), nil
}

// Hash calculates a hash from a byte slice using sha2-256 for the hashing algorithm
func Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
