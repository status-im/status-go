package reliability

import (
	"github.com/pkg/errors"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// ErrSDSManagerUnavailable means SDS could not run at all, as opposed to a
// payload simply not being SDS-wrapped.
var ErrSDSManagerUnavailable = errors.New("sds reliability manager unavailable")

type sdsManagerFactoryFunc func(*zap.Logger, sds.PersistenceCallbacks) (*sds.ReliabilityManager, error)

// MarkDependenciesMet must be called outside SDS callbacks, on the recovery worker.
func (r *Reliability) MarkDependenciesMet(messageIDs []string, channelID string) error {
	r.sdsOpsMu.RLock()
	defer r.sdsOpsMu.RUnlock()
	if r.sdsManager == nil {
		return ErrSDSManagerUnavailable
	}
	ids := make([]sds.MessageID, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = sds.MessageID(id)
	}
	return r.sdsManager.MarkDependenciesMet(ids, channelID)
}

func newSdsReliabilityManager(
	sdsManagerFactory sdsManagerFactoryFunc,
	logger *zap.Logger,
	persistence sds.PersistenceCallbacks,
	onMessageSent func(messageId sds.MessageID, channelId string),
	onMissingDependencies func(messageId sds.MessageID, missingDeps []sds.HistoryEntry, channelId string),
	retrievalHintProvider func(messageId sds.MessageID, channelId string) []byte,
) (*sds.ReliabilityManager, error) {
	reliabilityManager, err := sdsManagerFactory(logger, persistence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ReliabilityManager")
	}

	callbacks := sds.EventCallbacks{
		OnMessageSent: func(messageId sds.MessageID, channelId string) {
			logger.Debug("message sent with sds", zap.String("messageId", string(messageId)), zap.String("channelId", channelId))
			if onMessageSent != nil {
				onMessageSent(messageId, channelId)
			}
		},
		OnMessageExpired: func(messageId sds.MessageID, channelId string) {
			logger.Debug("unacknowledged sds message expired",
				zap.String("messageId", string(messageId)),
				zap.String("channelId", channelId))
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
		ChannelRetrievalHintProvider: func(messageId sds.MessageID, channelId string) []byte {
			if retrievalHintProvider == nil {
				return nil
			}

			hint := retrievalHintProvider(messageId, channelId)
			logger.Debug("resolved retrieval hint for sds message",
				zap.String("messageId", string(messageId)),
				zap.String("channelId", channelId),
				zap.Int("hintLen", len(hint)))

			return hint
		},
	}
	reliabilityManager.RegisterCallbacks(callbacks)

	return reliabilityManager, nil
}

// Wrap message with SDS protocol https://github.com/vacp2p/rfc-index/blob/main/vac/raw/sds.md
func (r *Reliability) WrapPayloadForSDS(payload []byte, channelID string) ([]byte, []byte, error) {
	// Keep the channel set for legacy retrieval-hint providers that lack it.
	r.sdsOpsMu.Lock()
	defer r.sdsOpsMu.Unlock()

	manager := r.sdsManager
	if manager == nil {
		return nil, nil, ErrSDSManagerUnavailable
	}
	r.sdsChannelMu.Lock()
	r.activeSDSChannelID = channelID
	r.sdsChannelMu.Unlock()
	defer func() {
		r.sdsChannelMu.Lock()
		r.activeSDSChannelID = ""
		r.sdsChannelMu.Unlock()
	}()

	sdsMessageID := crypto.Keccak256(payload)

	r.logger.Debug("original payload wrapped with SDS",
		zap.String("channelId", channelID),
		zap.Int("payloadLength", len(payload)),
		zap.String("messageId", cryptotypes.EncodeHex(sdsMessageID)),
	)
	sdsWrappedPayload, err := manager.WrapOutgoingMessage(payload, sds.MessageID(cryptotypes.EncodeHex(sdsMessageID)), channelID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to wrap message with SDS")
	}

	return sdsWrappedPayload, sdsMessageID, nil
}

// EnsureSDSChannel loads persisted state for a channel before it is ticked.
func (r *Reliability) EnsureSDSChannel(channelID string) error {
	r.sdsOpsMu.RLock()
	defer r.sdsOpsMu.RUnlock()

	if r.sdsManager == nil {
		return ErrSDSManagerUnavailable
	}
	return r.sdsManager.EnsureChannel(channelID)
}

// TickSDS returns due sync payloads, with generation errors reported per channel.
func (r *Reliability) TickSDS() ([]sds.TickEvent, error) {
	r.sdsOpsMu.RLock()
	defer r.sdsOpsMu.RUnlock()

	if r.sdsManager == nil {
		return nil, ErrSDSManagerUnavailable
	}
	return r.sdsManager.Tick()
}

func (r *Reliability) UnwrapPayloadFromSDS(wrappedPayload []byte) ([]byte, error) {
	payload, _, _, _, err := r.UnwrapPayloadFromSDSWithMetadata(wrappedPayload)
	return payload, err
}

func (r *Reliability) UnwrapPayloadFromSDSWithMetadata(wrappedPayload []byte) (
	payload []byte,
	messageID []byte,
	channelID string,
	isSDS bool,
	err error,
) {
	return r.UnwrapPayloadFromSDSWithContext(wrappedPayload, nil)
}

func (r *Reliability) UnwrapPayloadFromSDSWithContext(wrappedPayload, validationContext []byte) (
	payload []byte,
	messageID []byte,
	channelID string,
	isSDS bool,
	err error,
) {
	r.sdsOpsMu.RLock()
	defer r.sdsOpsMu.RUnlock()

	manager := r.sdsManager
	if manager == nil {
		return nil, nil, "", false, ErrSDSManagerUnavailable
	}

	unwrappedMessage, err := manager.UnwrapReceivedMessageWithContext(wrappedPayload, validationContext)
	if err != nil {
		r.logger.Debug("failed to unwrap received message with SDS", zap.Error(err))
		if errors.Is(err, sds.ErrNotSDS) {
			// Legacy messages are not necessarily SDS-wrapped.
			return wrappedPayload, nil, "", false, nil
		}
		return nil, nil, "", false, errors.Wrap(err, "failed to unwrap message with SDS")
	}

	if unwrappedMessage == nil {
		return nil, nil, "", false, errors.New("SDS unwrap returned no result")
	}
	if unwrappedMessage.ChannelId == nil || *unwrappedMessage.ChannelId == "" {
		return nil, nil, "", false, errors.New("SDS unwrap returned no channel ID")
	}

	channelID = *unwrappedMessage.ChannelId
	if unwrappedMessage.Message == nil || *unwrappedMessage.Message == nil {
		return nil, nil, channelID, true, nil
	}

	var missingDeps []sds.HistoryEntry
	if unwrappedMessage.MissingDeps != nil {
		missingDeps = *unwrappedMessage.MissingDeps
	}
	if len(missingDeps) > 0 {
		r.logger.Debug("missing deps with SDS", zap.Any("missing-deps", missingDeps))
	}

	payload = *unwrappedMessage.Message
	return payload, crypto.Keccak256(payload), channelID, true, nil
}
