package rendezvous

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	RecordsPrefix byte = 1 + iota

	TopicBodyDelimiter = 0xff
)

type RegistrationRecord struct {
	Id       peer.ID
	Addrs    [][]byte
	Ns       string
	Ttl      int
	Deadline time.Time
}

// TopicPart looks for TopicBodyDelimiter and returns topic prefix from the same key.
// It doesn't allocate memory for topic prefix.
func TopicPart(key []byte) []byte {
	idx := bytes.IndexByte(key, TopicBodyDelimiter)
	if idx == -1 {
		return nil
	}
	return key[1:idx] // first byte is RecordsPrefix
}

type RecordsKey []byte

func NewRecordsKey(ns string, id peer.ID) RecordsKey {
	key := make(RecordsKey, 2+len([]byte(ns))+len(id))
	key[0] = RecordsPrefix
	copy(key[1:], []byte(ns))
	key[1+len([]byte(ns))] = TopicBodyDelimiter
	copy(key[2+len([]byte(ns)):], id)
	return key
}

func (k RecordsKey) SamePrefix(prefix []byte) bool {
	return bytes.Equal(k[:len(prefix)], prefix)
}

func (k RecordsKey) String() string {
	return string(k)
}

// NewStorage creates instance of the storage.
func NewStorage(db *leveldb.DB) Storage {
	return Storage{
		db: db,
	}
}

// Storage manages records.
type Storage struct {
	db *leveldb.DB
}

// Add stores record using specified topic.
func (s Storage) Add(ns string, id peer.ID, addrs [][]byte, ttl int, deadline time.Time) (string, error) {
	key := NewRecordsKey(ns, id)
	stored := RegistrationRecord{
		Id:       id,
		Addrs:    addrs,
		Ttl:      ttl,
		Ns:       ns,
		Deadline: deadline,
	}

	var data bytes.Buffer
	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(stored)
	if err != nil {
		return "", err
	}
	return key.String(), s.db.Put(key, data.Bytes(), nil)
}

// RemoveBykey removes record from storage.
func (s *Storage) RemoveByKey(key string) error {
	return s.db.Delete([]byte(key), nil)
}

func (s *Storage) IterateAllKeys(iterator func(key RecordsKey, Deadline time.Time) error) error {
	iter := s.db.NewIterator(util.BytesPrefix([]byte{RecordsPrefix}), nil)
	defer iter.Release()

	for iter.Next() {
		var stored RegistrationRecord
		data := bytes.NewBuffer(iter.Value())
		decoder := gob.NewDecoder(data)
		if err := decoder.Decode(&stored); err != nil {
			return err
		}
		if err := iterator(RecordsKey(iter.Key()), stored.Deadline); err != nil {
			return err
		}
	}
	return nil
}

// GetRandom reads random records for specified topic up to specified limit.
func (s *Storage) GetRandom(ns string, limit int64) (rst []RegistrationRecord, err error) {
	prefixlen := 1 + len([]byte(ns))
	key := make(RecordsKey, prefixlen+32)
	key[0] = RecordsPrefix
	copy(key[1:], []byte(ns))
	key[prefixlen] = TopicBodyDelimiter
	prefixlen++

	iter := s.db.NewIterator(util.BytesPrefix(key[:prefixlen]), nil)
	defer iter.Release()
	uids := map[string]struct{}{}
	// it might be too much cause we do crypto/rand.Read. requires profiling
	for i := int64(0); i < limit*limit && len(rst) < int(limit); i++ {
		if _, err := rand.Read(key[prefixlen:]); err != nil {
			return nil, err
		}
		iter.Seek(key)
		for _, f := range []func() bool{iter.Prev, iter.Next} {
			if f() && key.SamePrefix(iter.Key()[:prefixlen]) {
				var stored RegistrationRecord
				data := bytes.NewBuffer(iter.Value())
				decoder := gob.NewDecoder(data)
				if err = decoder.Decode(&stored); err != nil {
					return nil, err
				}
				k := iter.Key()
				if _, exist := uids[string(k)]; exist {
					continue
				}
				uids[string(k)] = struct{}{}
				rst = append(rst, stored)
				break
			}
		}
	}
	return rst, nil
}
