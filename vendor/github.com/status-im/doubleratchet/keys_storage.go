package doubleratchet

import (
	"bytes"
	"sort"
)

// KeysStorage is an interface of an abstract in-memory or persistent keys storage.
type KeysStorage interface {
	// Get returns a message key by the given key and message number.
	Get(k Key, msgNum uint) (mk Key, ok bool, err error)

	// Put saves the given mk under the specified key and msgNum.
	Put(sessionID []byte, k Key, msgNum uint, mk Key, keySeqNum uint) error

	// DeleteMk ensures there's no message key under the specified key and msgNum.
	DeleteMk(k Key, msgNum uint) error

	// DeleteOldMKeys deletes old message keys for a session.
	DeleteOldMks(sessionID []byte, deleteUntilSeqKey uint) error

	// TruncateMks truncates the number of keys to maxKeys.
	TruncateMks(sessionID []byte, maxKeys int) error

	// Count returns number of message keys stored under the specified key.
	Count(k Key) (uint, error)

	// All returns all the keys
	All() (map[Key]map[uint]Key, error)
}

// KeysStorageInMemory is an in-memory message keys storage.
type KeysStorageInMemory struct {
	keys map[Key]map[uint]InMemoryKey
}

// Get returns a message key by the given key and message number.
func (s *KeysStorageInMemory) Get(pubKey Key, msgNum uint) (Key, bool, error) {
	if s.keys == nil {
		return Key{}, false, nil
	}
	msgs, ok := s.keys[pubKey]
	if !ok {
		return Key{}, false, nil
	}
	mk, ok := msgs[msgNum]
	if !ok {
		return Key{}, false, nil
	}
	return mk.messageKey, true, nil
}

type InMemoryKey struct {
	messageKey Key
	seqNum     uint
	sessionID  []byte
}

// Put saves the given mk under the specified key and msgNum.
func (s *KeysStorageInMemory) Put(sessionID []byte, pubKey Key, msgNum uint, mk Key, seqNum uint) error {
	if s.keys == nil {
		s.keys = make(map[Key]map[uint]InMemoryKey)
	}
	if _, ok := s.keys[pubKey]; !ok {
		s.keys[pubKey] = make(map[uint]InMemoryKey)
	}
	s.keys[pubKey][msgNum] = InMemoryKey{
		sessionID:  sessionID,
		messageKey: mk,
		seqNum:     seqNum,
	}
	return nil
}

// DeleteMk ensures there's no message key under the specified key and msgNum.
func (s *KeysStorageInMemory) DeleteMk(pubKey Key, msgNum uint) error {
	if s.keys == nil {
		return nil
	}
	if _, ok := s.keys[pubKey]; !ok {
		return nil
	}
	if _, ok := s.keys[pubKey][msgNum]; !ok {
		return nil
	}
	delete(s.keys[pubKey], msgNum)
	if len(s.keys[pubKey]) == 0 {
		delete(s.keys, pubKey)
	}
	return nil
}

// TruncateMks truncates the number of keys to maxKeys.
func (s *KeysStorageInMemory) TruncateMks(sessionID []byte, maxKeys int) error {
	var seqNos []uint
	// Collect all seq numbers
	for _, keys := range s.keys {
		for _, inMemoryKey := range keys {
			if bytes.Equal(inMemoryKey.sessionID, sessionID) {
				seqNos = append(seqNos, inMemoryKey.seqNum)
			}
		}
	}

	// Nothing to do if we haven't reached the limit
	if len(seqNos) <= maxKeys {
		return nil
	}

	// Take the sequence numbers we care about
	sort.Slice(seqNos, func(i, j int) bool { return seqNos[i] < seqNos[j] })
	toDeleteSlice := seqNos[:len(seqNos)-maxKeys]

	// Put in map for easier lookup
	toDelete := make(map[uint]bool)

	for _, seqNo := range toDeleteSlice {
		toDelete[seqNo] = true
	}

	for pubKey, keys := range s.keys {
		for i, inMemoryKey := range keys {
			if toDelete[inMemoryKey.seqNum] && bytes.Equal(inMemoryKey.sessionID, sessionID) {
				delete(s.keys[pubKey], i)
			}
		}
	}

	return nil
}

// DeleteOldMKeys deletes old message keys for a session.
func (s *KeysStorageInMemory) DeleteOldMks(sessionID []byte, deleteUntilSeqKey uint) error {
	for pubKey, keys := range s.keys {
		for i, inMemoryKey := range keys {
			if inMemoryKey.seqNum <= deleteUntilSeqKey && bytes.Equal(inMemoryKey.sessionID, sessionID) {
				delete(s.keys[pubKey], i)
			}
		}
	}
	return nil
}

// Count returns number of message keys stored under the specified key.
func (s *KeysStorageInMemory) Count(pubKey Key) (uint, error) {
	if s.keys == nil {
		return 0, nil
	}
	return uint(len(s.keys[pubKey])), nil
}

// All returns all the keys
func (s *KeysStorageInMemory) All() (map[Key]map[uint]Key, error) {
	response := make(map[Key]map[uint]Key)

	for pubKey, keys := range s.keys {
		response[pubKey] = make(map[uint]Key)
		for n, key := range keys {
			response[pubKey][n] = key.messageKey
		}
	}

	return response, nil
}
