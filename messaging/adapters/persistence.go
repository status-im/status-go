package adapters

import (
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
)

type KeysPersistence struct {
	P types.Persistence
}

var _ transport.KeysPersistence = (*KeysPersistence)(nil)

func (kp *KeysPersistence) All() (map[string][]byte, error) {
	return kp.P.WakuKeys()
}

func (kp *KeysPersistence) Add(chatID string, key []byte) error {
	return kp.P.AddWakuKey(chatID, key)
}

type ProcessedMessageIDsCache struct {
	P types.Persistence
}

var _ transport.ProcessedMessageIDsCachePersistence = (*ProcessedMessageIDsCache)(nil)

func (pm *ProcessedMessageIDsCache) Clear() error {
	return pm.P.MessageCacheClear()
}
func (pm *ProcessedMessageIDsCache) Hits(ids []string) (map[string]bool, error) {
	return pm.P.MessageCacheHits(ids)
}
func (pm *ProcessedMessageIDsCache) Add(ids []string, timestamp uint64) error {
	return pm.P.MessageCacheAdd(ids, timestamp)
}
func (pm *ProcessedMessageIDsCache) Clean(timestamp uint64) error {
	return pm.P.MessageCacheClearOlderThan(timestamp)
}
