package pb

import (
	"crypto/sha256"

	proto "github.com/golang/protobuf/proto"
)

// Hash calculates the hash of a waku message
func (msg *WakuMessage) Hash() ([]byte, error) {
	out, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return Hash(out), nil
}

// Hash calculates a hash from a byte slice using sha2-256 for the hashing algorithm
func Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
