package types

type Persistence interface {
	WakuKeys() (map[string][]byte, error)
	AddWakuKey(chatID string, key []byte) error

	MessageCacheAdd(ids []string, timestamp uint64) error
	MessageCacheClear() error
	MessageCacheClearOlderThan(timestamp uint64) error
	MessageCacheHits(ids []string) (map[string]bool, error)
}
