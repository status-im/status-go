package messaging

import "github.com/status-im/status-go/messaging/transport"

type keysPersistenceAdapter struct {
	p Persistence
}

var _ transport.KeysPersistence = (*keysPersistenceAdapter)(nil)

func (kp *keysPersistenceAdapter) All() (map[string][]byte, error) {
	return kp.p.WakuKeys()
}

func (kp *keysPersistenceAdapter) Add(chatID string, key []byte) error {
	return kp.p.AddWakuKey(chatID, key)
}

type processedMessageIDsCacheAdapter struct {
	p Persistence
}

var _ transport.ProcessedMessageIDsCachePersistence = (*processedMessageIDsCacheAdapter)(nil)

func (pm *processedMessageIDsCacheAdapter) Clear() error {
	return pm.p.MessageCacheClear()
}
func (pm *processedMessageIDsCacheAdapter) Hits(ids []string) (map[string]bool, error) {
	return pm.p.MessageCacheHits(ids)
}
func (pm *processedMessageIDsCacheAdapter) Add(ids []string, timestamp uint64) error {
	return pm.p.MessageCacheAdd(ids, timestamp)
}
func (pm *processedMessageIDsCacheAdapter) Clean(timestamp uint64) error {
	return pm.p.MessageCacheClearOlderThan(timestamp)
}
