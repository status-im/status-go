package datasync

import (
	"crypto/ecdsa"
	"github.com/golang/protobuf/proto"
	datasyncpeer "github.com/status-im/status-protocol-go/datasync/peer"
	datasyncnode "github.com/vacp2p/mvds/node"
	datasyncproto "github.com/vacp2p/mvds/protobuf"
	datasynctransport "github.com/vacp2p/mvds/transport"
	"go.uber.org/zap"
)

type DataSync struct {
	*datasyncnode.Node
	// DataSyncNodeTransport is the implemntation of the datasync transport interface
	*DataSyncNodeTransport
	logger         *zap.Logger
	sendingEnabled bool
}

func New(node *datasyncnode.Node, transport *DataSyncNodeTransport, sendingEnabled bool, logger *zap.Logger) *DataSync {
	return &DataSync{Node: node, DataSyncNodeTransport: transport, sendingEnabled: sendingEnabled, logger: logger}
}

func (d *DataSync) Add(publicKey *ecdsa.PublicKey, datasyncMessage datasyncproto.Payload) {
	packet := datasynctransport.Packet{
		Sender:  datasyncpeer.PublicKeyToPeerID(*publicKey),
		Payload: datasyncMessage,
	}
	d.DataSyncNodeTransport.AddPacket(packet)
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
			//copiedMessage := statusMessage.Copy()
			//copiedMessage.DataSyncLayerInfo.Payload = message.Body
			payloads = append(payloads, message.Body)
		}
		if d.sendingEnabled {
			d.Add(sender, datasyncMessage)
		}
	}

	return payloads
}

func unwrap(payload []byte) (datasyncPayload datasyncproto.Payload, err error) {
	err = proto.Unmarshal(payload, &datasyncPayload)
	return
}

func (d *DataSync) Stop() {
	d.Node.Stop()
}
