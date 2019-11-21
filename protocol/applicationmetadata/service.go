package applicationmetadata

import (
	"github.com/golang/protobuf/proto"
)

//go:generate protoc --go_out=. ./message.proto

func Unmarshal(payload []byte) (*Message, error) {
	var message Message
	err := proto.Unmarshal(payload, &message)
	if err != nil {
		return nil, err
	}

	return &message, nil
}
