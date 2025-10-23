package common

import (
	"crypto/ecdsa"

	"github.com/pkg/errors"

	mvdsnode "github.com/status-im/mvds/node"
	datasyncproto "github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"

	"github.com/status-im/status-go/crypto"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	messagingtypes "github.com/status-im/status-go/messaging/types"
)

var errReliabilityNotStarted = errors.New("reliability not started")

func (s *MessageSender) StartReliability(statusChangeEvent chan mvdsnode.PeerStatusChangeEvent, dispatcher func(peer state.PeerID, payload *datasyncproto.Payload) error) error {
	return s.reliability.Start(statusChangeEvent, dispatcher)
}

func (s *MessageSender) StopReliability() {
	s.reliability.Stop()
}

func (s *MessageSender) sendWithReliability(recipient *ecdsa.PublicKey, messageID cryptotypes.HexBytes, message []byte) error {
	if !s.reliability.Started() {
		return errReliabilityNotStarted
	}

	// No need to call transport tracking.
	// It is done in a data sync dispatch step.
	datasyncID, err := s.reliability.WrapAndQueueMessageForDispatch(recipient, message)
	if err != nil {
		return err
	}
	// We don't need to receive confirmations from our own devices
	if !crypto.IsPubKeyEqual(recipient, &s.identity.PublicKey) {
		confirmation := &messagingtypes.RawMessageConfirmation{
			DataSyncID: datasyncID[:],
			MessageID:  messageID,
			PublicKey:  crypto.CompressPubkey(recipient),
		}

		err = s.persistence.InsertPendingConfirmation(confirmation)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleReliabilityLayer tries to unwrap message as datasync one and in case of success
// returns cloned messages with replaced payloads
func (s *MessageSender) handleReliabilityLayer(m *messagingtypes.Message) ([]*messagingtypes.Message, [][]byte, error) {
	if !s.reliability.Started() {
		return nil, nil, errReliabilityNotStarted
	}

	datasyncMessage, err := s.reliability.UnwrapAndAcknowledgeMessage(
		m.SigPubKey(),
		m.EncryptionLayer.Payload,
	)
	if err != nil {
		return nil, nil, err
	}

	var statusMessages []*messagingtypes.Message
	for _, ds := range datasyncMessage.Messages {
		message, err := m.Clone()
		if err != nil {
			return nil, nil, err
		}
		message.EncryptionLayer.Payload = ds.Body
		statusMessages = append(statusMessages, message)
	}

	return statusMessages, datasyncMessage.Acks, nil
}
