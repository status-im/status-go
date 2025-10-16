package common

import (
	"encoding/hex"
	"sync"

	"github.com/jinzhu/copier"

	"github.com/status-im/status-go/messaging/types"
)

// MessageSenderPersistenceInMemory is an incomplete in-memory implementation
// of MessageSenderPersistence for testing purpose of MessageSender.
type MessageSenderPersistenceInMemory struct {
	mu sync.Mutex

	hashRatchetMessages        map[string]*types.ReceivedMessage   // hash -> received message
	hashRatchetMessagesByKeyID map[string][]*types.ReceivedMessage // keyID -> received messages
}

var _ types.MessageSenderPersistence = (*MessageSenderPersistenceInMemory)(nil)

func NewMessageSenderPersistenceInMemory() *MessageSenderPersistenceInMemory {
	p := &MessageSenderPersistenceInMemory{
		hashRatchetMessages:        make(map[string]*types.ReceivedMessage),
		hashRatchetMessagesByKeyID: make(map[string][]*types.ReceivedMessage),
	}

	return p
}

func (s *MessageSenderPersistenceInMemory) InsertPendingConfirmation(*types.RawMessageConfirmation) error {
	return nil
}

func (s *MessageSenderPersistenceInMemory) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error {
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

func (s *MessageSenderPersistenceInMemory) GetHashRatchetMessages(keyID []byte) ([]*types.ReceivedMessage, error) {
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

func (s *MessageSenderPersistenceInMemory) DeleteHashRatchetMessages(ids [][]byte) error {
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

func (s *MessageSenderPersistenceInMemory) DeleteHashRatchetMessagesOlderThan(timestamp int64) error {
	return nil
}
