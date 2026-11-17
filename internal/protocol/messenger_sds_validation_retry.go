package protocol

import (
	"bytes"
	"time"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/types"
	"go.uber.org/zap"
)

const (
	maxSDSValidationRetryEnvelopes = 1000
	maxSDSValidationRetryBytes     = 32 << 20
	sdsValidationRetryTTL          = 24 * time.Hour
)

type pendingSDSValidationEnvelope struct {
	filter    *types.ChatFilter
	message   *types.ReceivedMessage
	hash      string
	firstSeen time.Time
	size      int
}

func (m *Messenger) enqueueSDSValidationRetry(filter *types.ChatFilter, message *types.ReceivedMessage) {
	if filter == nil || message == nil || len(message.Hash) == 0 {
		m.logger.Warn("cannot retain SDS validation failure without envelope identity")
		return
	}

	hash := cryptotypes.EncodeHex(message.Hash)
	if _, exists := m.sdsValidationRetryHashes[hash]; exists {
		return
	}
	size := len(message.Sig) + 4 + len(message.Payload) +
		len(message.Padding) + len(message.Hash) + len(message.Dst) + len(message.ThirdPartyID)
	if size == 0 || len(m.sdsValidationRetries) >= maxSDSValidationRetryEnvelopes ||
		m.sdsValidationRetryBytes+size > maxSDSValidationRetryBytes {
		m.logger.Warn("SDS validation retry queue is full; dropping envelope",
			zap.String("hash", hash), zap.Int("queued", len(m.sdsValidationRetries)),
			zap.Int("queuedBytes", m.sdsValidationRetryBytes))
		delete(m.sdsValidationRetrySince, hash)
		return
	}

	now := time.Now()
	firstSeen, exists := m.sdsValidationRetrySince[hash]
	if !exists {
		firstSeen = now
	}
	if now.Sub(firstSeen) > sdsValidationRetryTTL {
		m.logger.Warn("SDS validation retry expired; dropping envelope", zap.String("hash", hash))
		delete(m.sdsValidationRetrySince, hash)
		return
	}

	if m.sdsValidationRetryHashes == nil {
		m.sdsValidationRetryHashes = make(map[string]struct{})
	}
	if m.sdsValidationRetrySince == nil {
		m.sdsValidationRetrySince = make(map[string]time.Time)
	}
	m.sdsValidationRetrySince[hash] = firstSeen
	m.sdsValidationRetryHashes[hash] = struct{}{}
	m.sdsValidationRetryBytes += size
	m.sdsValidationRetries = append(m.sdsValidationRetries, pendingSDSValidationEnvelope{
		filter:    filter,
		message:   cloneReceivedMessage(message),
		hash:      hash,
		firstSeen: firstSeen,
		size:      size,
	})
}

func (m *Messenger) takeSDSValidationRetries(messages map[types.ChatFilter][]*types.ReceivedMessage) map[types.ChatFilter][]*types.ReceivedMessage {
	if len(m.sdsValidationRetries) == 0 {
		return messages
	}
	if messages == nil {
		messages = make(map[types.ChatFilter][]*types.ReceivedMessage)
	}

	now := time.Now()
	for _, pending := range m.sdsValidationRetries {
		delete(m.sdsValidationRetryHashes, pending.hash)
		m.sdsValidationRetryBytes -= pending.size
		if now.Sub(pending.firstSeen) > sdsValidationRetryTTL {
			m.logger.Warn("SDS validation retry expired; dropping envelope",
				zap.String("hash", pending.hash))
			delete(m.sdsValidationRetrySince, pending.hash)
			continue
		}
		if containsEnvelopeHash(messages[*pending.filter], pending.hash) {
			continue
		}
		messages[*pending.filter] = append(messages[*pending.filter], pending.message)
	}
	m.sdsValidationRetries = nil
	return messages
}

func (m *Messenger) clearSDSValidationRetry(message *types.ReceivedMessage) {
	if message == nil || len(message.Hash) == 0 {
		return
	}
	delete(m.sdsValidationRetrySince, cryptotypes.EncodeHex(message.Hash))
}

func containsEnvelopeHash(messages []*types.ReceivedMessage, hash string) bool {
	for _, message := range messages {
		if cryptotypes.EncodeHex(message.Hash) == hash {
			return true
		}
	}
	return false
}

func cloneReceivedMessage(message *types.ReceivedMessage) *types.ReceivedMessage {
	clone := *message
	clone.Sig = bytes.Clone(message.Sig)
	clone.Payload = bytes.Clone(message.Payload)
	clone.Padding = bytes.Clone(message.Padding)
	clone.Hash = bytes.Clone(message.Hash)
	clone.Dst = bytes.Clone(message.Dst)
	return &clone
}
