package reliability

import (
	"crypto/ecdsa"

	"github.com/cockroachdb/errors"
	"go.uber.org/zap"

	mvdsnode "github.com/status-im/mvds/node"
	mvdsproto "github.com/status-im/mvds/protobuf"
	mvdsstate "github.com/status-im/mvds/state"

	"github.com/status-im/status-go/messaging/layers/reliability/datasync"
	datasyncpeer "github.com/status-im/status-go/messaging/layers/reliability/datasync/peer"
)

type MVDSDispatcher func(peer mvdsstate.PeerID, payload *mvdsproto.Payload) error

type Reliability struct {
	identity            *ecdsa.PrivateKey
	datasync            *datasync.DataSync
	datasyncPersistence mvdsnode.Persistence
	logger              *zap.Logger
}

func NewReliability(datasyncPersistence mvdsnode.Persistence, identity *ecdsa.PrivateKey, logger *zap.Logger) *Reliability {
	return &Reliability{
		identity:            identity,
		datasyncPersistence: datasyncPersistence,
		logger:              logger.Named("reliability"),
	}
}

func (r *Reliability) Start(statusChangeEvent chan mvdsnode.PeerStatusChangeEvent, dispatch MVDSDispatcher) error {
	dataSyncTransport := datasync.NewNodeTransport()
	dataSyncNode, err := mvdsnode.NewPersistentNode(
		r.datasyncPersistence,
		dataSyncTransport,
		datasyncpeer.PublicKeyToPeerID(r.identity.PublicKey),
		mvdsnode.BATCH,
		datasync.CalculateSendTime,
		statusChangeEvent,
		r.logger,
	)
	if err != nil {
		return err
	}

	r.datasync = datasync.New(dataSyncNode, dataSyncTransport, true, r.logger)

	r.datasync.Init(dispatch, r.logger)
	r.datasync.Start(datasync.DatasyncTicker)

	return nil
}

func (r *Reliability) Stop() {
	if r.Started() {
		r.datasync.Stop()
	}
	r.datasync = nil
}

func (r *Reliability) Started() bool {
	return r.datasync != nil
}

// WrapAndQueueMessageForDispatch wraps the message in the reliability layer,
// then queues it for delivery to the target public key using configured MVDSDispatcher.
func (r *Reliability) WrapAndQueueMessageForDispatch(publicKey *ecdsa.PublicKey, message []byte) (mvdsstate.MessageID, error) {
	groupID := datasync.ToOneToOneGroupID(&r.identity.PublicKey, publicKey)
	peerID := datasyncpeer.PublicKeyToPeerID(*publicKey)
	exist, err := r.datasync.IsPeerInGroup(groupID, peerID)
	if err != nil {
		return mvdsstate.MessageID{}, errors.Wrap(err, "failed to check if peer is in group")
	}
	if !exist {
		if err := r.datasync.AddPeer(groupID, peerID); err != nil {
			return mvdsstate.MessageID{}, errors.Wrap(err, "failed to add peer")
		}
	}
	return r.datasync.AppendMessage(groupID, message)
}

// UnwrapAndAcknowledge tries to unwrap received datasync message,
// and potentially acknowledges it.
func (r *Reliability) UnwrapAndAcknowledgeMessage(publicKey *ecdsa.PublicKey, message []byte) (*mvdsproto.Payload, error) {
	return r.datasync.Unwrap(publicKey, message)
}
