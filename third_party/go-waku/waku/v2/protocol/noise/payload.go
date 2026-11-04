package noise

import (
	"errors"

	n "github.com/waku-org/go-noise"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"
)

const NoiseEncryption = 2

// DecodePayloadV2 decodes a WakuMessage to a PayloadV2
// Currently, this is just a wrapper over deserializePayloadV2 and encryption/decryption is done on top (no KeyInfo)
func DecodePayloadV2(message *pb.WakuMessage) (*n.PayloadV2, error) {
	if message.GetVersion() != NoiseEncryption {
		return nil, errors.New("wrong message version while decoding payload")
	}
	return n.DeserializePayloadV2(message.Payload)
}

// EncodePayloadV2 encodes a PayloadV2 to a WakuMessage
// Currently, this is just a wrapper over serializePayloadV2 and encryption/decryption is done on top (no KeyInfo)
func EncodePayloadV2(payload2 *n.PayloadV2) (*pb.WakuMessage, error) {
	serializedPayload2, err := payload2.Serialize()
	if err != nil {
		return nil, err
	}

	return &pb.WakuMessage{
		Payload: serializedPayload2,
		Version: proto.Uint32(NoiseEncryption),
	}, nil
}
