package doubleratchet

// KeysStorage is an interface of an abstract in-memory or persistent keys storage.
type KeysStorage interface {
	// Get returns a message key by the given key and message number.
	Get(k Key, msgNum uint) (mk Key, ok bool, err error)

	// Put saves the given mk under the specified key and msgNum.
	Put(k Key, msgNum uint, mk Key) error

	// DeleteMk ensures there's no message key under the specified key and msgNum.
	DeleteMk(k Key, msgNum uint) error

	// DeletePk ensures there's no message keys under the specified key.
	DeletePk(k Key) error

	// Count returns number of message keys stored under the specified key.
	Count(k Key) (uint, error)

	// All returns all the keys
	All() (map[Key]map[uint]Key, error)
}

// KeysStorageInMemory is an in-memory message keys storage.
type KeysStorageInMemory struct {
	keys map[Key]map[uint]Key
}

// See KeysStorage.
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
	return mk, true, nil
}

// See KeysStorage.
func (s *KeysStorageInMemory) Put(pubKey Key, msgNum uint, mk Key) error {
	if s.keys == nil {
		s.keys = make(map[Key]map[uint]Key)
	}
	if _, ok := s.keys[pubKey]; !ok {
		s.keys[pubKey] = make(map[uint]Key)
	}
	s.keys[pubKey][msgNum] = mk
	return nil
}

// See KeysStorage.
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

// See KeysStorage.
func (s *KeysStorageInMemory) DeletePk(pubKey Key) error {
	if s.keys == nil {
		return nil
	}
	if _, ok := s.keys[pubKey]; !ok {
		return nil
	}
	delete(s.keys, pubKey)
	return nil
}

// See KeysStorage.
func (s *KeysStorageInMemory) Count(pubKey Key) (uint, error) {
	if s.keys == nil {
		return 0, nil
	}
	return uint(len(s.keys[pubKey])), nil
}

// See KeysStorage.
func (s *KeysStorageInMemory) All() (map[Key]map[uint]Key, error) {
	return s.keys, nil
}
