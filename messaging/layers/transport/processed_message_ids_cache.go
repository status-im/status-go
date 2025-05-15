package transport

type ProcessedMessageIDsCachePersistence interface {
	Clear() error
	Hits(ids []string) (map[string]bool, error)
	Add(ids []string, timestamp uint64) error
	Clean(timestamp uint64) error
}
