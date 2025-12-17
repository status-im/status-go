package protocol

import (
	"crypto/ecdsa"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	// ErrInvalidDecodedValue means that the decoded message is of wrong type.
	// This might mean that the status message serialization tag changed.
	ErrInvalidDecodedValue = errors.New("invalid decoded value type")
)

// TimestampInMsFromTime returns a TimestampInMs from a time.Time instance.
func TimestampInMsFromTime(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}

// WrapIntoAppLayerMessage wraps a payload into a protobuf message and signs it if an identity is provided
func WrapIntoAppLayerMessage(payload []byte, messageType protobuf.ApplicationMetadataMessage_Type, identity *ecdsa.PrivateKey) ([]byte, error) {
	var signature []byte
	if identity != nil {
		var err error
		signature, err = crypto.Sign(crypto.Keccak256(payload), identity)
		if err != nil {
			return nil, err
		}
	}

	message := &protobuf.ApplicationMetadataMessage{
		Signature: signature,
		Type:      messageType,
		Payload:   payload}
	return proto.Marshal(message)
}
