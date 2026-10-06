package reliability

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// ErrSDSManagerUnavailable means SDS could not run at all, as opposed to a
// payload simply not being SDS-wrapped.
var ErrSDSManagerUnavailable = errors.New("sds reliability manager unavailable")

type sdsManagerFactoryFunc func(*zap.Logger) (*sds.ReliabilityManager, error)

func newSdsReliabilityManager(
	sdsManagerFactory sdsManagerFactoryFunc,
	logger *zap.Logger,
	onMessageSent func(messageId sds.MessageID, channelId string),
	onMissingDependencies func(messageId sds.MessageID, missingDeps []sds.HistoryEntry, channelId string),
	retrievalHintProvider func(messageId sds.MessageID) []byte,
) (*sds.ReliabilityManager, error) {
	reliabilityManager, err := sdsManagerFactory(logger)
	if err != nil {
		fmt.Println("[sds-debug] SDS manager creation failed", "error", err)
		return nil, errors.Wrap(err, "failed to create ReliabilityManager")
	}
	fmt.Println("[sds-debug] fresh SDS manager created")

	callbacks := sds.EventCallbacks{
		OnMessageSent: func(messageId sds.MessageID, channelId string) {
			logger.Debug("message sent with sds", zap.String("messageId", string(messageId)), zap.String("channelId", channelId))
			if onMessageSent != nil {
				onMessageSent(messageId, channelId)
			}
		},
		OnMissingDependencies: func(messageId sds.MessageID, missingDeps []sds.HistoryEntry, channelId string) {
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

	return reliabilityManager, nil
}

// Wrap message with SDS protocol https://github.com/vacp2p/rfc-index/blob/main/vac/raw/sds.md
func (r *Reliability) WrapPayloadForSDS(payload []byte, channelID string) ([]byte, []byte, error) {
	r.sdsOpsMu.RLock()
	defer r.sdsOpsMu.RUnlock()

	manager := r.sdsManager
	if manager == nil {
		return nil, nil, ErrSDSManagerUnavailable
	}

	sdsMessageID := crypto.Keccak256(payload)

	r.logger.Debug("original payload wrapped with SDS",
		zap.String("channelId", channelID),
		zap.Int("payloadLength", len(payload)),
		zap.String("messageId", cryptotypes.EncodeHex(sdsMessageID)),
	)
	sdsWrappedPayload, err := manager.WrapOutgoingMessage(payload, sds.MessageID(cryptotypes.EncodeHex(sdsMessageID)), channelID)
	if err != nil {
		fmt.Println("[sds-debug] outgoing SDS wrap failed", "channelID", channelID, "sdsID", cryptotypes.EncodeHex(sdsMessageID), "error", err)
		return nil, nil, errors.Wrap(err, "failed to wrap message with SDS")
	}
	fmt.Println("[sds-debug] outgoing SDS wrapped", "channelID", channelID, "sdsID", cryptotypes.EncodeHex(sdsMessageID), "payloadBytes", len(payload), "wrappedBytes", len(sdsWrappedPayload))

	return sdsWrappedPayload, sdsMessageID, nil
}

func (r *Reliability) UnwrapPayloadFromSDS(wrappedPayload []byte) ([]byte, error) {
	r.sdsOpsMu.RLock()
	defer r.sdsOpsMu.RUnlock()

	manager := r.sdsManager
	if manager == nil {
		return nil, ErrSDSManagerUnavailable
	}

	unwrappedMessage, err := manager.UnwrapReceivedMessage(wrappedPayload)
	if err != nil {
		fmt.Println("[sds-debug] SDS unwrap failed; passing original payload through", "payloadBytes", len(wrappedPayload), "error", err)
		r.logger.Debug("failed to unwrap received message with SDS", zap.Error(err))
		// return original payload since wrapping is not mandatory for all kinds of messages
		return wrappedPayload, nil
	}

	missingDeps := *unwrappedMessage.MissingDeps
	fmt.Println("[sds-debug] incoming SDS unwrapped", "wrappedBytes", len(wrappedPayload), "payloadBytes", len(*unwrappedMessage.Message), "missingDependencies", len(missingDeps))
	if len(missingDeps) > 0 {
		r.logger.Debug("missing deps with SDS", zap.Any("missing-deps", missingDeps))
	}

	return *unwrappedMessage.Message, nil
}
