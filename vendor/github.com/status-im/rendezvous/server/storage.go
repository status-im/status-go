package server

import (
	"bytes"
	"crypto/rand"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	RecordsPrefix byte = 1 + iota

	TopicBodyDelimiter = 0xff
)

type StorageRecord struct {
	ENR  enr.Record
	Time time.Time
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

func NewRecordsKey(topic string, record enr.Record) RecordsKey {
	key := make(RecordsKey, 2+len([]byte(topic))+len(enode.ValidSchemes.NodeAddr(&record)))
	key[0] = RecordsPrefix
	copy(key[1:], []byte(topic))
	key[1+len([]byte(topic))] = TopicBodyDelimiter
	copy(key[2+len([]byte(topic)):], enode.ValidSchemes.NodeAddr(&record))
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
	return Storage{db: db}
}

// Storage manages records.
type Storage struct {
	db *leveldb.DB
}

// Add stores record using specified topic.
func (s Storage) Add(topic string, record enr.Record, t time.Time) (string, error) {
	key := NewRecordsKey(topic, record)
	stored := StorageRecord{
		ENR:  record,
		Time: t,
	}
	data, err := rlp.EncodeToBytes(stored)
	if err != nil {
		return "", err
	}
	return key.String(), s.db.Put(key, data, nil)
}

// RemoveBykey removes record from storage.
func (s *Storage) RemoveByKey(key string) error {
	return s.db.Delete([]byte(key), nil)
}

func (s *Storage) IterateAllKeys(iterator func(key RecordsKey, ttl time.Time) error) error {
	iter := s.db.NewIterator(util.BytesPrefix([]byte{RecordsPrefix}), nil)
	defer iter.Release()
	for iter.Next() {
		var stored StorageRecord
		if err := rlp.DecodeBytes(iter.Value(), &stored); err != nil {
			return err
		}
		if err := iterator(RecordsKey(iter.Key()), stored.Time); err != nil {
			return err
		}
	}
	return nil
}

// GetRandom reads random records for specified topic up to specified limit.
func (s *Storage) GetRandom(topic string, limit uint) (rst []enr.Record, err error) {
	prefixlen := 1 + len([]byte(topic))
	key := make(RecordsKey, prefixlen+32)
	key[0] = RecordsPrefix
	copy(key[1:], []byte(topic))
	key[prefixlen] = TopicBodyDelimiter
	prefixlen++

	iter := s.db.NewIterator(util.BytesPrefix(key[:prefixlen]), nil)
	defer iter.Release()
	uids := map[string]struct{}{}
	// it might be too much cause we do crypto/rand.Read. requires profiling
	for i := uint(0); i < limit*limit && len(rst) < int(limit); i++ {
		if _, err := rand.Read(key[prefixlen:]); err != nil {
			return nil, err
		}
		iter.Seek(key)
		for _, f := range []func() bool{iter.Prev, iter.Next} {
			if f() && key.SamePrefix(iter.Key()[:prefixlen]) {
				var stored StorageRecord
				if err = rlp.DecodeBytes(iter.Value(), &stored); err != nil {
					return nil, err
				}
				k := iter.Key()
				if _, exist := uids[string(k)]; exist {
					continue
				}
				uids[string(k)] = struct{}{}
				rst = append(rst, stored.ENR)
				break
			}
		}
	}
	return rst, nil
}
