package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/jinzhu/copier"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// StubPersistence is an in-memory implementation of types.Persistence for testing.
type StubPersistence struct {
	mu sync.Mutex

	wakuKeys map[string][]byte

	messageCache map[string]uint64

	hashRatchetMessages        map[string]*types.ReceivedMessage   // hash -> received message
	hashRatchetMessagesByKeyID map[string][]*types.ReceivedMessage // keyID -> received messages

	messageSegments   map[string]map[string][]*types.SegmentMessage // hash+pubkey -> segments
	completedSegments map[string]struct{}                           // hash
}

func NewStubPersistence() *StubPersistence {
	return &StubPersistence{
		wakuKeys:                   make(map[string][]byte),
		messageCache:               make(map[string]uint64),
		hashRatchetMessages:        make(map[string]*types.ReceivedMessage),
		hashRatchetMessagesByKeyID: make(map[string][]*types.ReceivedMessage),
		messageSegments:            make(map[string]map[string][]*types.SegmentMessage),
		completedSegments:          make(map[string]struct{}),
	}
}

func (s *StubPersistence) WakuKeys() (map[string][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	copy := make(map[string][]byte, len(s.wakuKeys))
	err := copier.Copy(&copy, s.wakuKeys)
	if err != nil {
		return nil, err
	}

	return copy, nil
}

func (s *StubPersistence) AddWakuKey(chatID string, key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copy := make([]byte, 0, len(key))
	err := copier.Copy(&copy, key)
	if err != nil {
		return err
	}

	s.wakuKeys[chatID] = copy
	return nil
}

func (s *StubPersistence) MessageCacheAdd(ids []string, timestamp uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		s.messageCache[id] = timestamp
	}
	return nil
}

func (s *StubPersistence) MessageCacheClear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageCache = make(map[string]uint64)
	return nil
}

func (s *StubPersistence) MessageCacheClearOlderThan(timestamp uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, ts := range s.messageCache {
		if ts < timestamp {
			delete(s.messageCache, id)
		}
	}
	return nil
}

func (s *StubPersistence) MessageCacheHits(ids []string) (map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hits := make(map[string]bool)
	for _, id := range ids {
		_, ok := s.messageCache[id]
		hits[id] = ok
	}
	return hits, nil
}

func (s *StubPersistence) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copy := &types.ReceivedMessage{}
	err := copier.Copy(copy, m)
	if err != nil {
		return err
	}

	hash := hex.EncodeToString(copy.Hash)
	key := hex.EncodeToString(keyID)
	s.hashRatchetMessages[hash] = copy
	s.hashRatchetMessagesByKeyID[key] = append(s.hashRatchetMessagesByKeyID[key], copy)

	return nil
}

func (s *StubPersistence) GetHashRatchetMessages(keyID []byte) ([]*types.ReceivedMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := hex.EncodeToString(keyID)
	msgs := s.hashRatchetMessagesByKeyID[key]

	copy := make([]*types.ReceivedMessage, 0, len(msgs))
	err := copier.Copy(&copy, msgs)
	if err != nil {
		return nil, err
	}

	return copy, nil
}

func (s *StubPersistence) DeleteHashRatchetMessages(ids [][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		hash := hex.EncodeToString(id)
		msg, ok := s.hashRatchetMessages[hash]
		if ok {
			// Remove from hashRatchetMessagesByKeyID as well
			for key, arr := range s.hashRatchetMessagesByKeyID {
				for i, m := range arr {
					if m == msg {
						s.hashRatchetMessagesByKeyID[key] = append(arr[:i], arr[i+1:]...)
						break
					}
				}
			}
			delete(s.hashRatchetMessages, hash)
		}
	}
	return nil
}

func (s *StubPersistence) IsMessageAlreadyCompleted(hash []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.completedSegments[string(hash)]
	return exists, nil
}

func (s *StubPersistence) SaveMessageSegment(segment *types.SegmentMessage, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copy := &types.SegmentMessage{
		SegmentMessage: proto.Clone(segment.SegmentMessage).(*protobuf.SegmentMessage),
	}

	hash := string(segment.EntireMessageHash)
	pubKey := string(crypto.CompressPubkey(sigPubKey))
	if s.messageSegments[hash] == nil {
		s.messageSegments[hash] = make(map[string][]*types.SegmentMessage)
	}
	s.messageSegments[hash][pubKey] = append(s.messageSegments[hash][pubKey], copy)
	return nil
}

func (s *StubPersistence) GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*types.SegmentMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	segments := s.messageSegments[string(hash)][string(crypto.CompressPubkey(sigPubKey))]
	copy := make([]*types.SegmentMessage, 0, len(segments))
	for _, seg := range segments {
		cloned := &types.SegmentMessage{
			SegmentMessage: proto.Clone(seg.SegmentMessage).(*protobuf.SegmentMessage),
		}
		copy = append(copy, cloned)
	}

	// Sort segments: non-parity first, then by index, then by parity index
	sort.SliceStable(copy, func(i, j int) bool {
		si, sj := copy[i], copy[j]

		// Non-parity segments first
		if si.SegmentsCount == 0 && sj.SegmentsCount > 0 {
			return false
		}
		if si.SegmentsCount > 0 && sj.SegmentsCount == 0 {
			return true
		}

		if si.SegmentsCount > 0 {
			return si.Index < sj.Index
		}

		return si.ParitySegmentIndex < sj.ParitySegmentIndex
	})

	return copy, nil
}

func (s *StubPersistence) CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.completedSegments[string(hash)] = struct{}{}

	h := string(hash)
	pubKey := string(crypto.CompressPubkey(sigPubKey))
	if s.messageSegments[h] != nil {
		delete(s.messageSegments[h], pubKey)
		if len(s.messageSegments[h]) == 0 {
			delete(s.messageSegments, h)
		}
	}
	return nil
}

func (s *StubPersistence) DeleteHashRatchetMessagesOlderThan(timestamp int64) error {
	// Not implemented for stub
	return nil
}

func (s *StubPersistence) InsertPendingConfirmation(*types.RawMessageConfirmation) error {
	// Not implemented for stub
	return nil
}

func (s *StubPersistence) RemoveMessageSegmentsOlderThan(timestamp int64) error {
	// Not implemented for stub
	return nil
}

func (s *StubPersistence) RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error {
	// Not implemented for stub
	return nil
}
