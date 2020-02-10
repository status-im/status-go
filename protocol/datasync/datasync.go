package datasync

import (
	"crypto/ecdsa"

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

func (d *DataSync) Handle(sender *ecdsa.PublicKey, payload []byte) [][]byte {
	var payloads [][]byte
	logger := d.logger.With(zap.String("site", "Handle"))

	datasyncMessage, err := unwrap(payload)
	// If it failed to decode is not a protobuf message, if it successfully decoded but body is empty, is likedly a protobuf wrapped message
	if err != nil || !datasyncMessage.IsValid() {
		logger.Debug("handling non-datasync message", zap.Error(err), zap.Bool("datasyncMessage.IsValid()", datasyncMessage.IsValid()))
		// Not a datasync message, return unchanged
		payloads = append(payloads, payload)
	} else {
		logger.Debug("handling datasync message")
		// datasync message
		for _, message := range datasyncMessage.Messages {
			payloads = append(payloads, message.Body)
		}
		if d.sendingEnabled {
			d.add(sender, datasyncMessage)
		}
	}

	return payloads
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
