package datasync

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/golang/protobuf/proto"
	datasyncpeer "github.com/status-im/status-protocol-go/datasync/peer"
	"github.com/vacp2p/mvds/protobuf"
	"github.com/vacp2p/mvds/state"
	"github.com/vacp2p/mvds/transport"
)

var errNotInitialized = errors.New("Datasync transport not initialized")

type DataSyncNodeTransport struct {
	packets  chan transport.Packet
	dispatch func(context.Context, *ecdsa.PublicKey, []byte) error
}

func NewDataSyncNodeTransport() *DataSyncNodeTransport {
	return &DataSyncNodeTransport{
		packets: make(chan transport.Packet),
	}
}

func (t *DataSyncNodeTransport) Init(dispatch func(context.Context, *ecdsa.PublicKey, []byte) error) {
	t.dispatch = dispatch
}

func (t *DataSyncNodeTransport) AddPacket(p transport.Packet) {
	t.packets <- p
}

func (t *DataSyncNodeTransport) Watch() transport.Packet {
	return <-t.packets
}

func (t *DataSyncNodeTransport) Send(_ state.PeerID, peer state.PeerID, payload protobuf.Payload) error {
	if t.dispatch == nil {
		return errNotInitialized
	}

	data, err := proto.Marshal(&payload)
	if err != nil {
		return err
	}

	publicKey, err := datasyncpeer.PeerIDToPublicKey(peer)
	if err != nil {
		return err
	}

	return t.dispatch(context.TODO(), publicKey, data)
}

// CalculateSendTime calculates the next epoch
// at which a message should be sent.
func CalculateSendTime(count uint64, time int64) int64 {
	return time + int64(count*2) // @todo this should match that time is increased by whisper periods, aka we only retransmit the first time when a message has expired.
}
