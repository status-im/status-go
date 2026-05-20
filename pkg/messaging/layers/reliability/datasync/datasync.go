package datasync

import (
	"crypto/ecdsa"
	"errors"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	datasyncnode "github.com/status-im/mvds/node"
	"github.com/status-im/mvds/protobuf"
	datasynctransport "github.com/status-im/mvds/transport"
	"go.uber.org/zap"

	datasyncpeer "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync/peer"
)

type DataSync struct {
	*datasyncnode.Node
	// NodeTransport is the implementation of the datasync transport interface.
	*NodeTransport
	logger *zap.Logger
	// sendingEnabled gates whether Unwrap feeds received messages into the node
	// for acknowledgement. Read on the inbound-message goroutine, written by
	// reliability.SetPaused on the lifecycle goroutine — hence atomic.
	sendingEnabled atomic.Bool
}

func New(node *datasyncnode.Node, transport *NodeTransport, sendingEnabled bool, logger *zap.Logger) *DataSync {
	d := &DataSync{Node: node, NodeTransport: transport, logger: logger}
	d.sendingEnabled.Store(sendingEnabled)
	return d
}

// Unwrap tries to unwrap datasync message and passes back the message to datasync in order to acknowledge any potential message and mark messages as acknowledged
func (d *DataSync) Unwrap(sender *ecdsa.PublicKey, payload []byte) (*protobuf.Payload, error) {
	logger := d.logger.With(zap.String("site", "Handle"))

	datasyncMessage, err := unwrap(payload)
	// If it failed to decode is not a protobuf message, if it successfully decoded but body is empty, is likedly a protobuf wrapped message
	if err != nil {
		logger.Debug("Unwrapping datasync message failed", zap.Error(err))
		return nil, err
	} else if !datasyncMessage.IsValid() {
		return nil, errors.New("handling non-datasync message")
	} else {
		logger.Debug("handling datasync message")
		if d.sendingEnabled.Load() {
			d.add(sender, &datasyncMessage)
		}
	}

	return &datasyncMessage, nil
}

func (d *DataSync) Stop() {
	d.Node.Stop()
}

// SetSendingEnabled toggles whether Unwrap feeds received datasync messages into
// the node for acknowledgement. It is flipped to false around a node recreate
// (see reliability.SetPaused) so Unwrap short-circuits instead of pushing
// packets onto a transport whose consumer goroutine has exited — AddPacket is
// non-blocking and buffered, so it wouldn't hang, but those packets would be
// silently dropped during the recreate window.
func (d *DataSync) SetSendingEnabled(v bool) {
	d.sendingEnabled.Store(v)
}

func (d *DataSync) add(publicKey *ecdsa.PublicKey, datasyncMessage *protobuf.Payload) {
	packet := datasynctransport.Packet{
		Sender:  datasyncpeer.PublicKeyToPeerID(*publicKey),
		Payload: datasyncMessage,
	}
	d.NodeTransport.AddPacket(packet)
}

func unwrap(payload []byte) (datasyncPayload protobuf.Payload, err error) {
	err = proto.Unmarshal(payload, &datasyncPayload)
	return
}
