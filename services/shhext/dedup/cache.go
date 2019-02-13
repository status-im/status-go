package dedup

import (
	"time"

	"github.com/status-im/status-go/db"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/crypto/sha3"
)

// cache represents a cache of whisper messages with a limit of 2 days.
// the limit is counted from the time when the message was added to the cache.
type cache struct {
	db  *leveldb.DB
	now func() time.Time
}

func newCache(db *leveldb.DB) *cache {
	return &cache{db, time.Now}
}

func (d *cache) Has(filterID string, message *whisper.Message) (bool, error) {
	has, err := d.db.Has(d.KeyToday(filterID, message), nil)

	if err != nil {
		return false, err
	}
	if has {
		return true, nil
	}

	return d.db.Has(d.keyYesterday(filterID, message), nil)
}

func (d *cache) Put(filterID string, messages []*whisper.Message) error {
	batch := leveldb.Batch{}

	for _, msg := range messages {
		batch.Put(d.KeyToday(filterID, msg), []byte{})
	}

	err := d.db.Write(&batch, nil)
	if err != nil {
		return err
	}

	return d.cleanOldEntries()
}

func (d *cache) PutIDs(messageIDs [][]byte) error {
	batch := leveldb.Batch{}

	for _, id := range messageIDs {
		batch.Put(id, []byte{})
	}

	err := d.db.Write(&batch, nil)
	if err != nil {
		return err
	}

	return d.cleanOldEntries()
}

func (d *cache) cleanOldEntries() error {
	// Cleaning up everything that is older than 2 days
	// We are using the fact that leveldb can do prefix queries and that
	// the entries are sorted by keys.
	// Here, we are looking for all the keys that are between
	// 00000000.* and <yesterday's date>.*
	// e.g. (0000000.* -> 20180424.*)

	limit := d.yesterdayDateString()

	r := &util.Range{
		Start: db.Key(db.DeduplicatorCache, []byte("00000000")),
		Limit: db.Key(db.DeduplicatorCache, []byte(limit)),
	}

	batch := leveldb.Batch{}
	iter := d.db.NewIterator(r, nil)
	for iter.Next() {
		batch.Delete(iter.Key())
	}
	iter.Release()

	return d.db.Write(&batch, nil)
}

func (d *cache) keyYesterday(filterID string, message *whisper.Message) []byte {
	return prefixedKey(d.yesterdayDateString(), filterID, message)
}

func (d *cache) KeyToday(filterID string, message *whisper.Message) []byte {
	return prefixedKey(d.todayDateString(), filterID, message)
}

func (d *cache) todayDateString() string {
	return dateString(d.now())
}

func (d *cache) yesterdayDateString() string {
	now := d.now()
	yesterday := now.Add(-24 * time.Hour)
	return dateString(yesterday)
}

func dateString(t time.Time) string {
	// Layouts must use the reference time Mon Jan 2 15:04:05 MST 2006
	return t.Format("20060102")
}

func prefixedKey(date, filterID string, message *whisper.Message) []byte {
	return db.Key(db.DeduplicatorCache, []byte(date), []byte(filterID), key(message))
}

func key(message *whisper.Message) []byte {
	data := make([]byte, len(message.Payload)+len(message.Topic))
	copy(data[:], message.Payload)
	copy(data[len(message.Payload):], message.Topic[:])
	digest := sha3.Sum512(data)
	return digest[:]
}
