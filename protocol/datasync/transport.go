package datasync

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vacp2p/mvds/protobuf"
	"github.com/vacp2p/mvds/state"
	"github.com/vacp2p/mvds/transport"
	"go.uber.org/zap"

	datasyncpeer "github.com/status-im/status-go/protocol/datasync/peer"
)

const backoffInterval = 30

var errNotInitialized = errors.New("Datasync transport not initialized")
var DatasyncTicker = 300 * time.Millisecond

// It's easier to calculate nextEpoch if we consider seconds as a unit rather than
// 300 ms, so we multiply the result by the ratio
var offsetToSecond = uint64(time.Second / DatasyncTicker)

// payloadTagSize is the tag size for the protobuf.Payload message which is number of fields * 2 bytes
var payloadTagSize = 14

// timestampPayloadSize is the maximum size in bytes for the timestamp field (uint64)
var timestampPayloadSize = 10

type NodeTransport struct {
	packets        chan transport.Packet
	logger         *zap.Logger
	maxMessageSize uint32
	dispatch       func(context.Context, *ecdsa.PublicKey, []byte, *protobuf.Payload) error
}

func NewNodeTransport() *NodeTransport {
	return &NodeTransport{
		packets: make(chan transport.Packet),
	}
}

func (t *NodeTransport) Init(dispatch func(context.Context, *ecdsa.PublicKey, []byte, *protobuf.Payload) error, maxMessageSize uint32, logger *zap.Logger) {
	t.dispatch = dispatch
	t.maxMessageSize = maxMessageSize
	t.logger = logger
}

func (t *NodeTransport) AddPacket(p transport.Packet) {
	t.packets <- p
}

func (t *NodeTransport) Watch() transport.Packet {
	return <-t.packets
}

func (t *NodeTransport) Send(_ state.PeerID, peer state.PeerID, payload protobuf.Payload) error {
	if t.dispatch == nil {
		return errNotInitialized
	}

	payloads := splitPayloadInBatches(&payload, int(t.maxMessageSize))
	for _, payload := range payloads {

		if !payload.IsValid() {
			t.logger.Error("payload is invalid")
			continue
		}

		data, err := proto.Marshal(payload)
		if err != nil {
			t.logger.Error("failed to marshal payload")
			continue
		}

		publicKey, err := datasyncpeer.IDToPublicKey(peer)
		if err != nil {
			t.logger.Error("failed to conver id to public key", zap.Error(err))
			continue
		}
		// We don't return an error otherwise datasync will keep
		// re-trying sending at each epoch
		err = t.dispatch(context.Background(), publicKey, data, payload)
		if err != nil {
			t.logger.Error("failed to send message", zap.Error(err))
			continue
		}
	}

	return nil
}

func splitPayloadInBatches(payload *protobuf.Payload, maxSizeBytes int) []*protobuf.Payload {
	newPayload := &protobuf.Payload{}
	var response []*protobuf.Payload
	currentSize := payloadTagSize

	// this is not going to be 100% accurate, but should be fine in most cases, faster
	// than using proto.Size
	for _, ack := range payload.Acks {
		if len(ack)+currentSize+1 > maxSizeBytes {
			// We check if it's valid as it might be that the initial message
			// is too big, in this case we still batch it
			if newPayload.IsValid() {
				response = append(response, newPayload)
			}
			newPayload = &protobuf.Payload{Acks: [][]byte{ack}}
			currentSize = len(ack) + payloadTagSize + 1
		} else {
			newPayload.Acks = append(newPayload.Acks, ack)
			currentSize += len(ack)
		}
	}

	for _, offer := range payload.Offers {
		if len(offer)+currentSize+1 > maxSizeBytes {
			if newPayload.IsValid() {
				response = append(response, newPayload)
			}
			newPayload = &protobuf.Payload{Offers: [][]byte{offer}}
			currentSize = len(offer) + payloadTagSize + 1
		} else {
			newPayload.Offers = append(newPayload.Offers, offer)
			currentSize += len(offer)
		}
	}

	for _, request := range payload.Requests {
		if len(request)+currentSize+1 > maxSizeBytes {
			if newPayload.IsValid() {
				response = append(response, newPayload)
			}
			newPayload = &protobuf.Payload{Requests: [][]byte{request}}
			currentSize = len(request) + payloadTagSize + 1
		} else {
			newPayload.Requests = append(newPayload.Requests, request)
			currentSize += len(request)
		}
	}

	for _, message := range payload.Messages {
		// We add the body size, the length field for payload, the length field for group id,
		// the length of timestamp, body and groupid
		if currentSize+1+1+timestampPayloadSize+len(message.Body)+len(message.GroupId) > maxSizeBytes {
			if newPayload.IsValid() {
				response = append(response, newPayload)
			}
			newPayload = &protobuf.Payload{Messages: []*protobuf.Message{message}}
			currentSize = timestampPayloadSize + len(message.Body) + len(message.GroupId) + payloadTagSize + 1 + 1
		} else {
			newPayload.Messages = append(newPayload.Messages, message)
			currentSize += len(message.Body) + len(message.GroupId) + timestampPayloadSize
		}
	}

	if newPayload.IsValid() {
		response = append(response, newPayload)
	}
	return response
}

// CalculateSendTime calculates the next epoch
// at which a message should be sent.
// We randomize it a bit so that not all messages are sent on the same epoch
func CalculateSendTime(count uint64, time int64) int64 {
	return time + int64(uint64(math.Exp2(float64(count-1)))*backoffInterval*offsetToSecond) + int64(rand.Intn(30)) // nolint: gosec

}
