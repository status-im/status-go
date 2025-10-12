package transport

type KeysPersistence interface {
	All() (map[string][]byte, error)
	Add(chatID string, key []byte) error
}

type ProcessedMessageIDsCachePersistence interface {
	Clear() error
	Hits(ids []string) (map[string]bool, error)
	Add(ids []string, timestamp uint64) error
	Clean(timestamp uint64) error
}
