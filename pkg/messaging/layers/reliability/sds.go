package reliability

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// Hashes
// 0xb0bf45126fdcae4c3cb4d0fc8bca3e089f8113b85a56ae89e1ee751499488dd8

func newSdsReliabilityManager(
	logger *zap.Logger,
	onMissingDependencies func(messageId sds.MessageID, missingDeps []sds.HistoryEntry, channelId string),
	retrievalHintProvider func(messageId sds.MessageID) []byte,
) *sds.ReliabilityManager {
	reliabilityManager, err := sds.NewReliabilityManager(logger)
	if err != nil {
		logger.Error("failed to create ReliabilityManager", zap.Error(err))
		return nil
	}

	callbacks := sds.EventCallbacks{
		OnMessageSent: func(messageId sds.MessageID, channelId string) {
			logger.Debug("message sent with sds", zap.String("messageId", string(messageId)), zap.String("channelId", channelId))
		},
		OnMissingDependencies: func(messageId sds.MessageID, missingDeps []sds.HistoryEntry, channelId string) {
			fmt.Println("OnMissingDependencies: Message has missing dependencies", missingDeps)
			for _, dep := range missingDeps {
				fmt.Printf("Missing dependency: MessageID=%s, RetrievalHintLen=%d, RetrievalHintHex=%x\n", dep.MessageID, len(dep.RetrievalHint), dep.RetrievalHint)
			}

			logger.Debug("missing dependencies",
				zap.String("messageId", string(messageId)),
				zap.String("channelId", channelId),
				zap.Any("missingDeps", missingDeps))

			if onMissingDependencies != nil {
				onMissingDependencies(messageId, missingDeps, channelId)
			}
		},
		OnMessageReady: func(messageId sds.MessageID, channelId string) {
			logger.Debug("message ready",
				zap.String("messageId", string(messageId)),
				zap.String("channelId", channelId))
		},
		RetrievalHintProvider: func(messageId sds.MessageID) []byte {
			if retrievalHintProvider == nil {
				return nil
			}

			hint := retrievalHintProvider(messageId)
			logger.Debug("resolved retrieval hint for sds message",
				zap.String("messageId", string(messageId)),
				zap.Int("hintLen", len(hint)))

			return hint
		},
	}
	reliabilityManager.RegisterCallbacks(callbacks)

	return reliabilityManager
}

// Wrap message with SDS protocol https://github.com/vacp2p/rfc-index/blob/main/vac/raw/sds.md
func (r *Reliability) WrapPayloadForSDS(payload []byte, channelID string) ([]byte, error) {
	sdsMessageID := crypto.Keccak256(payload)

	r.logger.Debug("original payload wrapped with SDS",
		zap.String("channelId", channelID),
		zap.Int("payloadLength", len(payload)),
		zap.String("messageId", cryptotypes.EncodeHex(sdsMessageID)),
	)
	sdsWrappedPayload, err := r.sdsManager.WrapOutgoingMessage(payload, sds.MessageID(cryptotypes.EncodeHex(sdsMessageID)), channelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap message with SDS")
	}

	return sdsWrappedPayload, nil
}

func (r *Reliability) UnwrapPayloadFromSDS(wrappedPayload []byte) ([]byte, error) {
	unwrappedMessage, err := r.sdsManager.UnwrapReceivedMessage(wrappedPayload)
	if err != nil {
		r.logger.Debug("failed to unwrap received message with SDS", zap.Error(err))
		// return original payload since wrapping is not mandatory for all kinds of messages
		return wrappedPayload, nil
	}

	missingDeps := *unwrappedMessage.MissingDeps
	if len(missingDeps) > 0 {
		r.logger.Debug("missing deps with SDS", zap.Any("missing-deps", missingDeps))
	}

	return *unwrappedMessage.Message, nil
}
