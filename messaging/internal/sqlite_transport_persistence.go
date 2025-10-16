package internal

import (
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
)

type SQLiteTransportPersistence struct {
	*transport.SQLiteKeysPersistence
	*transport.SQLiteProcessedMessageIDsCachePersistence
}

var _ types.TransportPersistence = (*SQLiteTransportPersistence)(nil)

func (s *SQLiteTransportPersistence) Keys() (map[string][]byte, error) {
	return nil, nil
}

func (s *SQLiteTransportPersistence) AddKey(chatID string, key []byte) error {
	return nil
}

func (s *SQLiteTransportPersistence) MessageCacheAdd(ids []string, timestamp uint64) error {
	return nil
}

func (s *SQLiteTransportPersistence) MessageCacheClear() error {
	return nil
}

func (s *SQLiteTransportPersistence) MessageCacheClearOlderThan(timestamp uint64) error {
	return nil
}

func (s *SQLiteTransportPersistence) MessageCacheHits(ids []string) (map[string]bool, error) {
	return nil, nil
}
