package reliability

import (
	"github.com/pkg/errors"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
)

func newSdsReliabilityManager(transport *transport.Transport, logger *zap.Logger) *sds.ReliabilityManager {
	reliabilityManager, err := sds.NewReliabilityManager(logger)
	if err != nil {
		logger.Error("failed to create ReliabilityManager", zap.Error(err))
		return nil
	}

	callbacks := sds.EventCallbacks{
		OnMessageSent: func(messageId sds.MessageID, channelId string) {
			logger.Debug("message sent with sds", zap.String("messageId", string(messageId)), zap.String("channelId", channelId))
			transport.ConfirmMessageSent(string(messageId))
		},
		OnMissingDependencies: func(messageId sds.MessageID, missingDeps []sds.MessageID, channelId string) {
			logger.Debug("missing dependencies",
				zap.String("messageId", string(messageId)),
				zap.String("channelId", channelId),
				zap.Any("missingDeps", missingDeps))
		},
		OnMessageReady: func(messageId sds.MessageID, channelId string) {
			logger.Debug("message ready",
				zap.String("messageId", string(messageId)),
				zap.String("channelId", channelId))
		},
	}
	reliabilityManager.RegisterCallbacks(callbacks)

	return reliabilityManager
}

// Wrap message with SDS protocol https://github.com/vacp2p/rfc-index/blob/main/vac/raw/sds.md
func (r *Reliability) WrapPayloadForSDS(payload []byte, messageID string, communityID []byte) ([]byte, error) {
	r.logger.Debug("original payload wrapped with SDS",
		zap.String("channelId", cryptotypes.EncodeHex(communityID)),
		zap.Int("payloadLength", len(payload)),
		zap.String("messageId", messageID),
	)
	sdsWrappedPayload, err := r.sdsManager.WrapOutgoingMessage(payload, sds.MessageID(messageID), cryptotypes.EncodeHex(communityID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap a community message with SDS")
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
