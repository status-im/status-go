package sender

import (
	"context"
	"crypto/ecdsa"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

var errReliabilityNotStarted = errors.New("reliability not started")

func (s *Sender) scheduleReliableSend(recipient *ecdsa.PublicKey, message []byte) ([]byte, error) {
	if !s.stack.Reliability.Started() {
		return nil, errReliabilityNotStarted
	}

	datasyncID, err := s.stack.Reliability.WrapAndQueueMessageForDispatch(recipient, message)
	if err != nil {
		return nil, err
	}

	return datasyncID[:], nil
}

func (s *Sender) SendPrivateReliability(recipient *ecdsa.PublicKey, wrappedPayload []byte, messages [][]byte) error {
	if crypto.IsPubKeyEqual(recipient, &s.identity.PublicKey) {
		// Serialize this with scheduling the legacy route and its confirmation.
		s.installationSendMu.Lock()
		defer s.installationSendMu.Unlock()
		if len(messages) != 0 {
			return s.sendLegacySelfMessages(wrappedPayload, messages)
		}
	}
	spec, err := s.buildEncryptedMessage(s.identity, recipient, wrappedPayload)
	if err != nil {
		return errors.Wrap(err, "failed to build encrypted message")
	}
	return s.sendReliableEnvelope(recipient, spec, messages)
}

func (s *Sender) sendReliableEnvelope(recipient *ecdsa.PublicKey, spec *encryption.ProtocolMessageSpec, messages [][]byte) error {
	messageIDs := make([]cryptotypes.HexBytes, 0, len(messages))
	for _, msgPayload := range messages {
		messageIDs = append(messageIDs, messagingtypes.MessageID(&s.identity.PublicKey, msgPayload))
	}
	logger := s.logger.Named("sendPrivateReliability").With(
		zap.Stringers("messageIDs", messageIDs),
		zap.String("recipient", crypto.PubkeyToHex(recipient)),
	)
	payload, sharedSecretKey, err := s.processAndMarshalMessageSpec(spec)
	if err != nil {
		return errors.Wrap(err, "failed to process and marshal message spec")
	}

	hashes, wakuMessages, err := s.sendPrivate(context.Background(), logger, sendPrivateParams{
		recipient:       recipient,
		payload:         payload,
		sharedSecretKey: sharedSecretKey,
	})
	if err != nil {
		return errors.Wrap(err, "failed to send private message")
	}

	logger.Debug("sent-message",
		zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)),
	)

	byteMessageIDs := make([][]byte, len(messageIDs))
	for i, id := range messageIDs {
		byteMessageIDs[i] = []byte(id)
	}

	s.stack.Transport.TrackMany(byteMessageIDs, hashes, wakuMessages)

	pubsub.Publish(s.publisher, SentMessage{
		Private:                true,
		Recipient:              recipient,
		RecipientInstallations: spec.Installations,
		MessageIDs:             byteMessageIDs,
	})

	return nil
}
