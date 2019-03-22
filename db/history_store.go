package db

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

// NewHistoryStore returns HistoryStore instance.
func NewHistoryStore(db *leveldb.DB) HistoryStore {
	return HistoryStore{
		topicDB:   NewDBNamespace(db, TopicHistoryBucket),
		requestDB: NewDBNamespace(db, HistoryRequestBucket),
	}
}

// HistoryStore provides utility methods for quering history and requests store.
type HistoryStore struct {
	topicDB   DB
	requestDB DB
}

// GetHistory creates history instance and loads history from database.
// Returns instance populated with topic and duration if history is not found in database.
func (h HistoryStore) GetHistory(topic whisper.TopicType, duration time.Duration) (TopicHistory, error) {
	thist := h.NewHistory(topic, duration)
	err := thist.Load()
	if err != nil && err != errors.ErrNotFound {
		return TopicHistory{}, err
	}
	return thist, nil
}

// NewRequest returns instance of the HistoryRequest.
func (h HistoryStore) NewRequest() HistoryRequest {
	return HistoryRequest{requestDB: h.requestDB, topicDB: h.topicDB}
}

// NewHistory creates TopicHistory object with required values.
func (h HistoryStore) NewHistory(topic whisper.TopicType, duration time.Duration) TopicHistory {
	return TopicHistory{db: h.topicDB, Duration: duration, Topic: topic}
}

// GetRequest loads HistoryRequest from database.
func (h HistoryStore) GetRequest(id common.Hash) (HistoryRequest, error) {
	req := HistoryRequest{requestDB: h.requestDB, topicDB: h.topicDB, ID: id}
	err := req.Load()
	if err != nil {
		return HistoryRequest{}, err
	}
	return req, nil
}

// GetAllRequests loads all not-finished history requests from database.
func (h HistoryStore) GetAllRequests() ([]HistoryRequest, error) {
	rst := []HistoryRequest{}
	iter := h.requestDB.NewIterator(h.requestDB.Range(nil, nil))
	for iter.Next() {
		req := HistoryRequest{
			requestDB: h.requestDB,
			topicDB:   h.topicDB,
		}
		err := req.RawUnmarshall(iter.Value())
		if err != nil {
			return nil, err
		}
		rst = append(rst, req)
	}
	return rst, nil
}

// GetHistoriesByTopic returns all histories with a given topic.
// This is needed when we will have multiple range per single topic.
// TODO explain
func (h HistoryStore) GetHistoriesByTopic(topic whisper.TopicType) ([]TopicHistory, error) {
	rst := []TopicHistory{}
	iter := h.topicDB.NewIterator(h.topicDB.Range(topic[:], nil))
	for iter.Next() {
		key := TopicHistoryKey{}
		copy(key[:], iter.Key())
		th, err := LoadTopicHistoryFromKey(h.topicDB, key)
		if err != nil {
			return nil, err
		}
		rst = append(rst, th)
	}
	return rst, nil
}
