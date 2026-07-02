package wakuv2

import (
	"bytes"
	"context"
	"sync"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	gocommon "github.com/status-im/status-go/common"
)

// Defaults mirror go-waku's publish.MessageSentCheck so behaviour is unchanged.
const (
	defaultMaxHashQueryLength   = 50
	defaultHashQueryInterval    = 3 * time.Second
	defaultMessageSentPeriod    = 3  // seconds after publish before a message is queried against the store
	defaultMessageExpiredPeriod = 10 // seconds after publish before an unconfirmed message is considered expired
	defaultStoreQueryTimeout    = 30 * time.Second
)

// storenodePeerProvider supplies the store node MessageSentCheck queries. It is
// satisfied by the go-waku StorenodeCycle today; once the cycle is removed the
// same seam can be backed by the on-demand StoreClient, so the check needs no
// changes when that swap happens.
type storenodePeerProvider interface {
	GetActiveStorenodePeerInfo() peer.AddrInfo
}

// timeProvider is the subset of the node timesource MessageSentCheck needs.
type timeProvider interface {
	Now() time.Time
}

// MessageSentCheck is a Status-owned reimplementation of go-waku's
// publish.MessageSentCheck (it implements publish.ISentCheck so it can be
// plugged into the go-waku MessageSender). It tracks outgoing messages and,
// once messageSentPeriod has elapsed since publish, asks a store node whether
// each was stored: messages confirmed stored are reported on messageStoredChan,
// messages still missing after messageExpiredPeriod on messageExpiredChan.
//
// Unlike go-waku's version it does not depend on the concrete StorenodeCycle —
// the store node is supplied through the injectable storenodePeerProvider, so
// the check is decoupled from the cycle's lifecycle.
type MessageSentCheck struct {
	messageIDs           map[string]map[gethcommon.Hash]uint32
	messageIDsMu         sync.RWMutex
	messageStoredChan    chan gethcommon.Hash
	messageExpiredChan   chan gethcommon.Hash
	ctx                  context.Context
	messageVerifier      publish.StorenodeMessageVerifier
	peerProvider         storenodePeerProvider
	timesource           timeProvider
	logger               *zap.Logger
	maxHashQueryLength   uint64
	hashQueryInterval    time.Duration
	messageSentPeriod    uint32
	messageExpiredPeriod uint32
	storeQueryTimeout    time.Duration
}

var _ publish.ISentCheck = (*MessageSentCheck)(nil)

// NewMessageSentCheck creates a MessageSentCheck with go-waku's default timings.
func NewMessageSentCheck(
	ctx context.Context,
	messageVerifier publish.StorenodeMessageVerifier,
	peerProvider storenodePeerProvider,
	timesource timeProvider,
	msgStoredChan chan gethcommon.Hash,
	msgExpiredChan chan gethcommon.Hash,
	logger *zap.Logger,
) *MessageSentCheck {
	return &MessageSentCheck{
		messageIDs:           make(map[string]map[gethcommon.Hash]uint32),
		messageStoredChan:    msgStoredChan,
		messageExpiredChan:   msgExpiredChan,
		ctx:                  ctx,
		messageVerifier:      messageVerifier,
		peerProvider:         peerProvider,
		timesource:           timesource,
		logger:               logger,
		maxHashQueryLength:   defaultMaxHashQueryLength,
		hashQueryInterval:    defaultHashQueryInterval,
		messageSentPeriod:    defaultMessageSentPeriod,
		messageExpiredPeriod: defaultMessageExpiredPeriod,
		storeQueryTimeout:    defaultStoreQueryTimeout,
	}
}

// Add registers a published message to be confirmed against the store.
func (m *MessageSentCheck) Add(topic string, messageID gethcommon.Hash, sentTime uint32) {
	m.messageIDsMu.Lock()
	defer m.messageIDsMu.Unlock()

	if _, ok := m.messageIDs[topic]; !ok {
		m.messageIDs[topic] = make(map[gethcommon.Hash]uint32)
	}
	m.messageIDs[topic][messageID] = sentTime
}

// DeleteByMessageIDs stops tracking the given message ids, used by scenarios
// like a message being acknowledged out-of-band via MVDS.
func (m *MessageSentCheck) DeleteByMessageIDs(messageIDs []gethcommon.Hash) {
	m.messageIDsMu.Lock()
	defer m.messageIDsMu.Unlock()

	for pubsubTopic, subMsgs := range m.messageIDs {
		for _, hash := range messageIDs {
			delete(subMsgs, hash)
			if len(subMsgs) == 0 {
				delete(m.messageIDs, pubsubTopic)
			} else {
				m.messageIDs[pubsubTopic] = subMsgs
			}
		}
	}
}

// Start runs the periodic store-confirmation loop until the context is done. It
// is meant to be invoked in its own goroutine (the MessageSender does so).
func (m *MessageSentCheck) Start() {
	defer gocommon.LogOnPanic()
	ticker := time.NewTicker(m.hashQueryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.messageIDsMu.Lock()
			pubsubTopics := make([]string, 0, len(m.messageIDs))
			pubsubMessageIds := make([][]gethcommon.Hash, 0, len(m.messageIDs))
			pubsubMessageTime := make([][]uint32, 0, len(m.messageIDs))
			for pubsubTopic, subMsgs := range m.messageIDs {
				var queryMsgIds []gethcommon.Hash
				var queryMsgTime []uint32
				for msgID, sendTime := range subMsgs {
					if uint64(len(queryMsgIds)) >= m.maxHashQueryLength {
						break
					}
					// Only query messages whose sent grace period has elapsed.
					if m.timesource.Now().Unix() > int64(sendTime)+int64(m.messageSentPeriod) {
						queryMsgIds = append(queryMsgIds, msgID)
						queryMsgTime = append(queryMsgTime, sendTime)
					}
				}
				if len(queryMsgIds) > 0 {
					pubsubTopics = append(pubsubTopics, pubsubTopic)
					pubsubMessageIds = append(pubsubMessageIds, queryMsgIds)
					pubsubMessageTime = append(pubsubMessageTime, queryMsgTime)
				}
			}
			m.messageIDsMu.Unlock()

			pubsubProcessedMessages := make([][]gethcommon.Hash, len(pubsubTopics))
			for i, pubsubTopic := range pubsubTopics {
				pubsubProcessedMessages[i] = m.messageHashBasedQuery(m.ctx, pubsubMessageIds[i], pubsubMessageTime[i], pubsubTopic)
			}

			m.messageIDsMu.Lock()
			for i, pubsubTopic := range pubsubTopics {
				subMsgs, ok := m.messageIDs[pubsubTopic]
				if !ok {
					continue
				}
				for _, hash := range pubsubProcessedMessages[i] {
					delete(subMsgs, hash)
					if len(subMsgs) == 0 {
						delete(m.messageIDs, pubsubTopic)
					} else {
						m.messageIDs[pubsubTopic] = subMsgs
					}
				}
			}
			m.messageIDsMu.Unlock()
		}
	}
}

// messageHashBasedQuery asks the selected store node which of the given hashes it
// holds. Hashes that are stored are reported on messageStoredChan; hashes still
// missing past their expiry are reported on messageExpiredChan. Both sets are
// returned so the caller can stop tracking them.
func (m *MessageSentCheck) messageHashBasedQuery(ctx context.Context, hashes []gethcommon.Hash, relayTime []uint32, pubsubTopic string) []gethcommon.Hash {
	selectedPeer := m.peerProvider.GetActiveStorenodePeerInfo()
	if selectedPeer.ID == "" {
		m.logger.Error("no store peer id available", zap.String("pubsubTopic", pubsubTopic))
		return []gethcommon.Hash{}
	}

	requestID := protocol.GenerateRequestID()

	messageHashes := make([]pb.MessageHash, len(hashes))
	for i, hash := range hashes {
		messageHashes[i] = pb.ToMessageHash(hash.Bytes())
	}

	m.logger.Debug("store.queryByHash request", zap.String("requestID", hexutil.Encode(requestID)), zap.Stringer("peerID", selectedPeer.ID), zap.Stringers("messageHashes", messageHashes))

	queryCtx, cancel := context.WithTimeout(ctx, m.storeQueryTimeout)
	defer cancel()
	result, err := m.messageVerifier.MessageHashesExist(queryCtx, requestID, selectedPeer, m.maxHashQueryLength, messageHashes)
	if err != nil {
		m.logger.Error("store.queryByHash failed", zap.String("requestID", hexutil.Encode(requestID)), zap.Stringer("peerID", selectedPeer.ID), zap.Error(err))
		return []gethcommon.Hash{}
	}

	m.logger.Debug("store.queryByHash result", zap.String("requestID", hexutil.Encode(requestID)), zap.Int("messages", len(result)))

	var ackHashes []gethcommon.Hash
	var missedHashes []gethcommon.Hash
	for i, hash := range hashes {
		found := false
		for _, msgHash := range result {
			if bytes.Equal(msgHash.Bytes(), hash.Bytes()) {
				found = true
				break
			}
		}

		if found {
			ackHashes = append(ackHashes, hash)
			m.messageStoredChan <- hash
		}

		if !found && m.timesource.Now().Unix() > int64(relayTime[i])+int64(m.messageExpiredPeriod) {
			missedHashes = append(missedHashes, hash)
			m.messageExpiredChan <- hash
		}
	}

	return append(ackHashes, missedHashes...)
}
