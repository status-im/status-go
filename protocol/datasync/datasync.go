package datasync

import (
	"crypto/ecdsa"
	"errors"

	"github.com/golang/protobuf/proto"
	datasyncnode "github.com/vacp2p/mvds/node"
	datasyncproto "github.com/vacp2p/mvds/protobuf"
	datasynctransport "github.com/vacp2p/mvds/transport"
	"go.uber.org/zap"

	datasyncpeer "github.com/status-im/status-go/protocol/datasync/peer"
)

type DataSync struct {
	*datasyncnode.Node
	// NodeTransport is the implementation of the datasync transport interface.
	*NodeTransport
	logger         *zap.Logger
	sendingEnabled bool
}

func New(node *datasyncnode.Node, transport *NodeTransport, sendingEnabled bool, logger *zap.Logger) *DataSync {
	return &DataSync{Node: node, NodeTransport: transport, sendingEnabled: sendingEnabled, logger: logger}
}

// UnwrapPayloadsAndAcks tries to unwrap datasync message and return messages payloads
// and acknowledgements for previously sent messages
func (d *DataSync) UnwrapPayloadsAndAcks(sender *ecdsa.PublicKey, payload []byte) ([][]byte, [][]byte, error) {
	var payloads [][]byte
	var acks [][]byte
	logger := d.logger.With(zap.String("site", "Handle"))

	datasyncMessage, err := unwrap(payload)
	// If it failed to decode is not a protobuf message, if it successfully decoded but body is empty, is likedly a protobuf wrapped message
	if err != nil {
		logger.Debug("Unwrapping datasync message failed", zap.Error(err))
		return nil, nil, err
	} else if !datasyncMessage.IsValid() {
		return nil, nil, errors.New("handling non-datasync message")
	} else {
		logger.Debug("handling datasync message")
		// datasync message
		for _, message := range datasyncMessage.Messages {
			payloads = append(payloads, message.Body)
		}

		acks = append(acks, datasyncMessage.Acks...)

		if d.sendingEnabled {
			d.add(sender, datasyncMessage)
		}
	}

	return payloads, acks, nil
}

func (d *DataSync) Stop() {
	d.Node.Stop()
}

func (d *DataSync) add(publicKey *ecdsa.PublicKey, datasyncMessage datasyncproto.Payload) {
	packet := datasynctransport.Packet{
		Sender:  datasyncpeer.PublicKeyToPeerID(*publicKey),
		Payload: datasyncMessage,
	}
	d.NodeTransport.AddPacket(packet)
}

func unwrap(payload []byte) (datasyncPayload datasyncproto.Payload, err error) {
	err = proto.Unmarshal(payload, &datasyncPayload)
	return
}
