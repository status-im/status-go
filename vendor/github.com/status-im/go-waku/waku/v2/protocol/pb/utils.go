package pb

import (
	gcrypto "github.com/ethereum/go-ethereum/crypto"
	proto "github.com/golang/protobuf/proto"
)

func (msg *WakuMessage) Hash() ([]byte, error) {
	out, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return Hash(out), nil
}

func Hash(data []byte) []byte {
	return gcrypto.Keccak256(data)
}
