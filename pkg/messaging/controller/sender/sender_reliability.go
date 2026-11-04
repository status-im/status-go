package sender

import (
	"context"
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	mvdsproto "github.com/status-im/mvds/protobuf"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/diagnostics"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync"
	datasyncpeer "github.com/status-im/status-go/pkg/messaging/layers/reliability/datasync/peer"
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

	if diagnostics.Enabled() {
		diagnostics.Emit(diagnostics.Event{Metadata: diagnostics.Metadata{
			MessageIDs:     diagnostics.Hexes([][]byte{messagingtypes.MessageID(&s.identity.PublicKey, message)}),
			MVDSMessageIDs: diagnostics.Hexes([][]byte{datasyncID[:]}),
			Recipient:      crypto.PubkeyToHex(recipient), Reliability: "mvds",
		}, Stage: "mvds_message_queued"})
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
	metadata := diagnostics.Metadata{}
	if diagnostics.Enabled() {
		var mvdsPayload mvdsproto.Payload
		if err := proto.Unmarshal(wrappedPayload, &mvdsPayload); err == nil {
			metadata = datasync.PayloadTraceMetadata(datasyncpeer.PublicKeyToPeerID(*recipient), &mvdsPayload)
		}
		metadata.MessageIDs = diagnostics.Hexes(byteMessageIDs)
		metadata.Recipient = crypto.PubkeyToHex(recipient)
		metadata.Reliability = "mvds"
	}
	ctx, attempt := diagnostics.BeginWithMetadata(context.Background(), metadata)
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
