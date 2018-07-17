package doubleratchet

// KeysStorage is an interface of an abstract in-memory or persistent keys storage.
type KeysStorage interface {
	// Get returns a message key by the given key and message number.
	Get(k Key, msgNum uint) (mk Key, ok bool)

	// Put saves the given mk under the specified key and msgNum.
	Put(k Key, msgNum uint, mk Key)

	// DeleteMk ensures there's no message key under the specified key and msgNum.
	DeleteMk(k Key, msgNum uint)

	// DeletePk ensures there's no message keys under the specified key.
	DeletePk(k Key)

	// Count returns number of message keys stored under the specified key.
	Count(k Key) uint

	// All returns all the keys
	All() map[Key]map[uint]Key
}

// KeysStorageInMemory is an in-memory message keys storage.
type KeysStorageInMemory struct {
	keys map[Key]map[uint]Key
}

// See KeysStorage.
func (s *KeysStorageInMemory) Get(pubKey Key, msgNum uint) (Key, bool) {
	if s.keys == nil {
		return Key{}, false
	}
	msgs, ok := s.keys[pubKey]
	if !ok {
		return Key{}, false
	}
	mk, ok := msgs[msgNum]
	if !ok {
		return Key{}, false
	}
	return mk, true
}

// See KeysStorage.
func (s *KeysStorageInMemory) Put(pubKey Key, msgNum uint, mk Key) {
	if s.keys == nil {
		s.keys = make(map[Key]map[uint]Key)
	}
	if _, ok := s.keys[pubKey]; !ok {
		s.keys[pubKey] = make(map[uint]Key)
	}
	s.keys[pubKey][msgNum] = mk
}

// See KeysStorage.
func (s *KeysStorageInMemory) DeleteMk(pubKey Key, msgNum uint) {
	if s.keys == nil {
		return
	}
	if _, ok := s.keys[pubKey]; !ok {
		return
	}
	if _, ok := s.keys[pubKey][msgNum]; !ok {
		return
	}
	delete(s.keys[pubKey], msgNum)
	if len(s.keys[pubKey]) == 0 {
		delete(s.keys, pubKey)
	}
}

// See KeysStorage.
func (s *KeysStorageInMemory) DeletePk(pubKey Key) {
	if s.keys == nil {
		return
	}
	if _, ok := s.keys[pubKey]; !ok {
		return
	}
	delete(s.keys, pubKey)
}

// See KeysStorage.
func (s *KeysStorageInMemory) Count(pubKey Key) uint {
	if s.keys == nil {
		return 0
	}
	return uint(len(s.keys[pubKey]))
}

// See KeysStorage.
func (s *KeysStorageInMemory) All() map[Key]map[uint]Key {
	return s.keys
}
