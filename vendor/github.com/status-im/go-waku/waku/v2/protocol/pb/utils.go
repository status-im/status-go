package pb

import (
	gcrypto "github.com/ethereum/go-ethereum/crypto"
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

// Hash calculates a hash from a byte slice using keccak256 for the hashing algorithm
func Hash(data []byte) []byte {
	return gcrypto.Keccak256(data)
}
