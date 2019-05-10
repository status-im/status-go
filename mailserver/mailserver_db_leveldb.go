package mailserver

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/params"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"time"
)

type LevelDBImpl struct {
	// We can't embed as there are some state problems with go-routines
	ldb *leveldb.DB
}

type LevelDBIterator struct {
	iterator.Iterator
}

func (i *LevelDBIterator) DBKey() *DBKey {
	return &DBKey{
		raw: i.Key(),
	}
}

func (i *LevelDBIterator) GetEnvelope(bloom []byte) ([]byte, error) {
	var envelopeBloom []byte
	rawValue := make([]byte, len(i.Value()))
	copy(rawValue, i.Value())

	key := i.DBKey()
	if len(key.Bytes()) != DBKeyLength {
		var err error
		envelopeBloom, err = extractBloomFromEncodedEnvelope(rawValue)
		if err != nil {
			return nil, err
		}
	} else {
		envelopeBloom = whisper.TopicToBloom(key.Topic())
	}
	if !whisper.BloomFilterMatch(bloom, envelopeBloom) {
		return nil, nil
	}
	return rawValue, nil

}

func NewLevelDBImpl(config *params.WhisperConfig) (*LevelDBImpl, error) {
	// Open opens an existing leveldb database
	db, err := leveldb.OpenFile(config.DataDir, nil)
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		log.Info("database is corrupted trying to recover", "path", config.DataDir)
		db, err = leveldb.RecoverFile(config.DataDir, nil)
	}
	return &LevelDBImpl{ldb: db}, err
}

// Build iterator returns an iterator given a start/end and a cursor
func (db *LevelDBImpl) BuildIterator(query CursorQuery) Iterator {
	defer recoverLevelDBPanics("BuildIterator")

	i := db.ldb.NewIterator(&util.Range{Start: query.start, Limit: query.end}, nil)
	// seek to the end as we want to return envelopes in a descending order
	if len(query.cursor) == CursorLength {
		i.Seek(query.cursor)
	}
	return &LevelDBIterator{i}
}

// GetEnvelope get an envelope by its key
func (db *LevelDBImpl) GetEnvelope(key *DBKey) ([]byte, error) {
	defer recoverLevelDBPanics("GetEnvelope")

	return db.ldb.Get(key.Bytes(), nil)
}

// Prune removes envelopes older than time
func (db *LevelDBImpl) Prune(t time.Time, batchSize int) (int, error) {
	defer recoverLevelDBPanics("Prune")

	var zero common.Hash
	var emptyTopic whisper.TopicType
	kl := NewDBKey(0, emptyTopic, zero)
	ku := NewDBKey(uint32(t.Unix()), emptyTopic, zero)
	query := CursorQuery{
		start: kl.Bytes(),
		end:   ku.Bytes(),
	}
	i := db.BuildIterator(query)
	defer i.Release()

	batch := leveldb.Batch{}
	removed := 0

	for i.Next() {
		batch.Delete(i.DBKey().Bytes())

		if batch.Len() == batchSize {
			if err := db.ldb.Write(&batch, nil); err != nil {
				return removed, err
			}

			removed = removed + batch.Len()
			batch.Reset()
		}
	}

	if batch.Len() > 0 {
		if err := db.ldb.Write(&batch, nil); err != nil {
			return removed, err
		}

		removed = removed + batch.Len()
	}

	return removed, nil
}

// SaveEnvelope stores an envelope in leveldb and increments the metrics
func (db *LevelDBImpl) SaveEnvelope(env *whisper.Envelope) error {
	defer recoverLevelDBPanics("SaveEnvelope")

	key := NewDBKey(env.Expiry-env.TTL, env.Topic, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
		archivedErrorsCounter.Inc(1)
		return err
	}

	if err = db.ldb.Put(key.Bytes(), rawEnvelope, nil); err != nil {
		log.Error(fmt.Sprintf("Writing to DB failed: %s", err))
		archivedErrorsCounter.Inc(1)
	}
	archivedMeter.Mark(1)
	archivedSizeMeter.Mark(int64(whisper.EnvelopeHeaderLength + len(env.Data)))
	return err
}

func (db *LevelDBImpl) Close() error {
	return db.ldb.Close()
}

func recoverLevelDBPanics(calleMethodName string) {
	// Recover from possible goleveldb panics
	if r := recover(); r != nil {
		if errString, ok := r.(string); ok {
			log.Error(fmt.Sprintf("recovered from panic in %s: %s", calleMethodName, errString))
		}
	}
}
