package publish

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
)

const DefaultMaxHashQueryLength = 50
const DefaultHashQueryInterval = 3 * time.Second
const DefaultMessageSentPeriod = 3    // in seconds
const DefaultMessageExpiredPerid = 10 // in seconds

type MessageSentCheckOption func(*MessageSentCheck) error

type ISentCheck interface {
	Start()
	Add(topic string, messageID common.Hash, sentTime uint32)
	DeleteByMessageIDs(messageIDs []common.Hash)
	SetStorePeerID(peerID peer.ID)
}

// MessageSentCheck tracks the outgoing messages and check against store node
// if the message sent time has passed the `messageSentPeriod`, the message id will be includes for the next query
// if the message keeps missing after `messageExpiredPerid`, the message id will be expired
type MessageSentCheck struct {
	messageIDs          map[string]map[common.Hash]uint32
	messageIDsMu        sync.RWMutex
	storePeerID         peer.ID
	messageStoredChan   chan common.Hash
	messageExpiredChan  chan common.Hash
	ctx                 context.Context
	store               *store.WakuStore
	timesource          timesource.Timesource
	logger              *zap.Logger
	maxHashQueryLength  uint64
	hashQueryInterval   time.Duration
	messageSentPeriod   uint32
	messageExpiredPerid uint32
}

// NewMessageSentCheck creates a new instance of MessageSentCheck with default parameters
func NewMessageSentCheck(ctx context.Context, store *store.WakuStore, timesource timesource.Timesource, msgStoredChan chan common.Hash, msgExpiredChan chan common.Hash, logger *zap.Logger) *MessageSentCheck {
	return &MessageSentCheck{
		messageIDs:          make(map[string]map[common.Hash]uint32),
		messageIDsMu:        sync.RWMutex{},
		messageStoredChan:   msgStoredChan,
		messageExpiredChan:  msgExpiredChan,
		ctx:                 ctx,
		store:               store,
		timesource:          timesource,
		logger:              logger,
		maxHashQueryLength:  DefaultMaxHashQueryLength,
		hashQueryInterval:   DefaultHashQueryInterval,
		messageSentPeriod:   DefaultMessageSentPeriod,
		messageExpiredPerid: DefaultMessageExpiredPerid,
	}
}

// WithMaxHashQueryLength sets the maximum number of message hashes to query in one request
func WithMaxHashQueryLength(count uint64) MessageSentCheckOption {
	return func(params *MessageSentCheck) error {
		params.maxHashQueryLength = count
		return nil
	}
}

// WithHashQueryInterval sets the interval to query the store node
func WithHashQueryInterval(interval time.Duration) MessageSentCheckOption {
	return func(params *MessageSentCheck) error {
		params.hashQueryInterval = interval
		return nil
	}
}

// WithMessageSentPeriod sets the delay period to query the store node after message is published
func WithMessageSentPeriod(period uint32) MessageSentCheckOption {
	return func(params *MessageSentCheck) error {
		params.messageSentPeriod = period
		return nil
	}
}

// WithMessageExpiredPerid sets the period that a message is considered expired
func WithMessageExpiredPerid(period uint32) MessageSentCheckOption {
	return func(params *MessageSentCheck) error {
		params.messageExpiredPerid = period
		return nil
	}
}

// Add adds a message for message sent check
func (m *MessageSentCheck) Add(topic string, messageID common.Hash, sentTime uint32) {
	m.messageIDsMu.Lock()
	defer m.messageIDsMu.Unlock()

	if _, ok := m.messageIDs[topic]; !ok {
		m.messageIDs[topic] = make(map[common.Hash]uint32)
	}
	m.messageIDs[topic][messageID] = sentTime
}

// DeleteByMessageIDs deletes the message ids from the message sent check, used by scenarios like message acked with MVDS
func (m *MessageSentCheck) DeleteByMessageIDs(messageIDs []common.Hash) {
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

// SetStorePeerID sets the peer id of store node
func (m *MessageSentCheck) SetStorePeerID(peerID peer.ID) {
	m.storePeerID = peerID
}

// Start checks if the tracked outgoing messages are stored periodically
func (m *MessageSentCheck) Start() {
	ticker := time.NewTicker(m.hashQueryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			m.logger.Debug("stop the look for message stored check")
			return
		case <-ticker.C:
			m.messageIDsMu.Lock()
			m.logger.Debug("running loop for messages stored check", zap.Any("messageIds", m.messageIDs))
			pubsubTopics := make([]string, 0, len(m.messageIDs))
			pubsubMessageIds := make([][]common.Hash, 0, len(m.messageIDs))
			pubsubMessageTime := make([][]uint32, 0, len(m.messageIDs))
			for pubsubTopic, subMsgs := range m.messageIDs {
				var queryMsgIds []common.Hash
				var queryMsgTime []uint32
				for msgID, sendTime := range subMsgs {
					if uint64(len(queryMsgIds)) >= m.maxHashQueryLength {
						break
					}
					// message is sent 5 seconds ago, check if it's stored
					if uint32(m.timesource.Now().Unix()) > sendTime+m.messageSentPeriod {
						queryMsgIds = append(queryMsgIds, msgID)
						queryMsgTime = append(queryMsgTime, sendTime)
					}
				}
				m.logger.Debug("store query for message hashes", zap.Any("queryMsgIds", queryMsgIds), zap.String("pubsubTopic", pubsubTopic))
				if len(queryMsgIds) > 0 {
					pubsubTopics = append(pubsubTopics, pubsubTopic)
					pubsubMessageIds = append(pubsubMessageIds, queryMsgIds)
					pubsubMessageTime = append(pubsubMessageTime, queryMsgTime)
				}
			}
			m.messageIDsMu.Unlock()

			pubsubProcessedMessages := make([][]common.Hash, len(pubsubTopics))
			for i, pubsubTopic := range pubsubTopics {
				processedMessages := m.messageHashBasedQuery(m.ctx, pubsubMessageIds[i], pubsubMessageTime[i], pubsubTopic)
				pubsubProcessedMessages[i] = processedMessages
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
			m.logger.Debug("messages for next store hash query", zap.Any("messageIds", m.messageIDs))
			m.messageIDsMu.Unlock()

		}
	}
}

func (m *MessageSentCheck) messageHashBasedQuery(ctx context.Context, hashes []common.Hash, relayTime []uint32, pubsubTopic string) []common.Hash {
	selectedPeer := m.storePeerID
	if selectedPeer == "" {
		m.logger.Error("no store peer id available", zap.String("pubsubTopic", pubsubTopic))
		return []common.Hash{}
	}

	var opts []store.RequestOption
	requestID := protocol.GenerateRequestID()
	opts = append(opts, store.WithRequestID(requestID))
	opts = append(opts, store.WithPeer(selectedPeer))
	opts = append(opts, store.WithPaging(false, m.maxHashQueryLength))
	opts = append(opts, store.IncludeData(false))

	messageHashes := make([]pb.MessageHash, len(hashes))
	for i, hash := range hashes {
		messageHashes[i] = pb.ToMessageHash(hash.Bytes())
	}

	m.logger.Debug("store.queryByHash request", zap.String("requestID", hexutil.Encode(requestID)), zap.Stringer("peerID", selectedPeer), zap.Stringers("messageHashes", messageHashes))

	result, err := m.store.QueryByHash(ctx, messageHashes, opts...)
	if err != nil {
		m.logger.Error("store.queryByHash failed", zap.String("requestID", hexutil.Encode(requestID)), zap.Stringer("peerID", selectedPeer), zap.Error(err))
		return []common.Hash{}
	}

	m.logger.Debug("store.queryByHash result", zap.String("requestID", hexutil.Encode(requestID)), zap.Int("messages", len(result.Messages())))

	var ackHashes []common.Hash
	var missedHashes []common.Hash
	for i, hash := range hashes {
		found := false
		for _, msg := range result.Messages() {
			if bytes.Equal(msg.GetMessageHash(), hash.Bytes()) {
				found = true
				break
			}
		}

		if found {
			ackHashes = append(ackHashes, hash)
			m.messageStoredChan <- hash
		}

		if !found && uint32(m.timesource.Now().Unix()) > relayTime[i]+m.messageExpiredPerid {
			missedHashes = append(missedHashes, hash)
			m.messageExpiredChan <- hash
		}
	}

	m.logger.Debug("ack message hashes", zap.Stringers("ackHashes", ackHashes))
	m.logger.Debug("missed message hashes", zap.Stringers("missedHashes", missedHashes))

	return append(ackHashes, missedHashes...)
}
