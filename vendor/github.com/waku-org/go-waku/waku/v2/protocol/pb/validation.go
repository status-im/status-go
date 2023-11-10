package pb

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

const MaxMetaAttrLength = 64

var (
	errMissingPayload      = errors.New("missing Payload field")
	errMissingContentTopic = errors.New("missing ContentTopic field")
	errInvalidMetaLength   = errors.New("invalid length for Meta field")
)

func (msg *WakuMessage) Validate() error {
	if len(msg.Payload) == 0 {
		return errMissingPayload
	}

	if msg.ContentTopic == "" {
		return errMissingContentTopic
	}

	if len(msg.Meta) > MaxMetaAttrLength {
		return errInvalidMetaLength
	}

	return nil
}

func Unmarshal(data []byte) (*WakuMessage, error) {
	msg := &WakuMessage{}
	err := proto.Unmarshal(data, msg)
	if err != nil {
		return nil, err
	}

	err = msg.Validate()
	if err != nil {
		return nil, err
	}

	return msg, nil

}
