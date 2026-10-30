package sender

import (
	"context"
	"crypto/ecdsa"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/diagnostics"
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

func (s *Sender) SendPrivateReliability(recipient *ecdsa.PublicKey, wrappedPayload []byte, messages [][]byte) (sendErr error) {
	messageIDs := make([]cryptotypes.HexBytes, 0, len(messages))
	for _, msgPayload := range messages {
		messageIDs = append(messageIDs, messagingtypes.MessageID(&s.identity.PublicKey, msgPayload))
	}
	byteMessageIDs := make([][]byte, len(messageIDs))
	for i, id := range messageIDs {
		byteMessageIDs[i] = []byte(id)
	}
	ctx, attempt := diagnostics.Begin(context.Background(), byteMessageIDs, crypto.PubkeyToHex(recipient), "mvds")
	defer func() {
		if sendErr != nil {
			diagnostics.Emit(diagnostics.Event{Metadata: attempt, Stage: "attempt_failed", Error: sendErr.Error()})
		}
	}()

	logger := s.logger.Named("sendPrivateReliability").With(
		zap.Stringers("messageIDs", messageIDs),
		zap.String("recipient", crypto.PubkeyToHex(recipient)),
	)

	spec, err := s.buildEncryptedMessage(s.identity, recipient, wrappedPayload)
	if err != nil {
		return errors.Wrap(err, "failed to build encrypted message")
	}

	payload, sharedSecretKey, err := s.processAndMarshalMessageSpec(spec)
	if err != nil {
		return errors.Wrap(err, "failed to process and marshal message spec")
	}

	hashes, wakuMessages, err := s.sendPrivate(ctx, logger, sendPrivateParams{
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

	if diagnostics.Enabled() {
		diagnostics.Emit(diagnostics.Event{Metadata: attempt, Stage: "dispatch", EnvelopeHashes: diagnostics.Hexes(hashes)})
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
