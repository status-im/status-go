package common

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/crypto"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	messagingtypes "github.com/status-im/status-go/messaging/types"
)

var errReliabilityNotStarted = errors.New("reliability not started")

func (s *MessageSender) StartReliability() error {
	dispatcher := func(publicKey *ecdsa.PublicKey, wrappedPayload []byte, messages [][]byte) error {
		messageIDs := make([]cryptotypes.HexBytes, 0, len(messages))
		for _, msgPayload := range messages {
			messageIDs = append(messageIDs, messagingtypes.MessageID(&s.identity.PublicKey, msgPayload))
		}

		err := s.sendPrivateEncryptedMessage(context.Background(), s.identity, publicKey, wrappedPayload, messageIDs)
		if err != nil {
			return err
		}

		return nil
	}

	return s.reliability.Start(dispatcher)
}

func (s *MessageSender) StopReliability() {
	s.reliability.Stop()
}

func (s *MessageSender) ReportUserOnline(publicKey *ecdsa.PublicKey, eventTime uint64) {
	s.reliability.ReportPeerOnline(publicKey, eventTime)
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
func (s *MessageSender) handleReliabilityLayer(m *messagingtypes.Message) ([]*messagingtypes.Message, []cryptotypes.HexBytes, error) {
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

	ackedMessageIDs := make([]cryptotypes.HexBytes, 0, len(datasyncMessage.Acks))
	for _, ack := range datasyncMessage.Acks {
		messageIDBytes, err := s.markAsConfirmed(ack, true)
		if err != nil {
			s.logger.Info("got datasync acknowledge for message we don't have in db", zap.String("ack", hex.EncodeToString(ack)))
			continue
		}

		s.transport.ConfirmMessageDelivered(messageIDBytes.String())

		ackedMessageIDs = append(ackedMessageIDs, messageIDBytes)
	}

	return statusMessages, ackedMessageIDs, nil
}
