package datasync

import (
	"crypto/ecdsa"
	"errors"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	datasyncnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	datasynctransport "github.com/status-im/mvds/transport"
	"go.uber.org/zap"

	datasyncpeer "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync/peer"
)

type DataSync struct {
	*datasyncnode.Node
	// NodeTransport is the implementation of the datasync transport interface.
	*NodeTransport
	logger         *zap.Logger
	sendingEnabled atomic.Bool
}

func New(node *datasyncnode.Node, transport *NodeTransport, sendingEnabled bool, logger *zap.Logger) *DataSync {
	d := &DataSync{Node: node, NodeTransport: transport, logger: logger}
	d.sendingEnabled.Store(sendingEnabled)
	return d
}

// Unwrap tries to unwrap datasync message and passes back the message to datasync in order to acknowledge any potential message and mark messages as acknowledged
func (d *DataSync) Unwrap(sender *ecdsa.PublicKey, payload []byte) (*protobuf.Payload, error) {
	if sender == nil {
		return nil, errors.New("MVDS message has no sender")
	}
	datasyncMessage, err := DecodePayload(payload)
	if err != nil {
		return nil, err
	}
	return d.ProcessPayload(datasyncpeer.PublicKeyToPeerID(*sender), &datasyncMessage)
}

func (d *DataSync) ProcessPayload(sender state.PeerID, datasyncMessage *protobuf.Payload) (*protobuf.Payload, error) {
	logger := d.logger.With(zap.String("site", "Handle"))
	if !datasyncMessage.IsValid() {
		return nil, errors.New("handling non-datasync message")
	}
	logger.Debug("handling datasync message")
	if d.sendingEnabled.Load() {
		d.AddPacket(datasynctransport.Packet{Sender: sender, Payload: datasyncMessage})
	}
	return datasyncMessage, nil
}

func (d *DataSync) Stop() {
	d.Node.Stop()
}

func (d *DataSync) SetSendingEnabled(v bool) {
	d.sendingEnabled.Store(v)
}

func DecodePayload(payload []byte) (datasyncPayload protobuf.Payload, err error) {
	err = proto.Unmarshal(payload, &datasyncPayload)
	return
}
