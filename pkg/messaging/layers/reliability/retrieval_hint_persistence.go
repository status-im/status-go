package reliability

// SDSRetrievalHintPersistence stores transport hashes associated with SDS
// message IDs, bounded by SDS history. Received-message records separately
// certify complete reconstruction and unwrap, and survive SDS history eviction
// for types.StoreMessageRetention from the first receipt; duplicates do not extend it.
// Outgoing retrieval hints and merely routed envelopes are not such evidence.
type SDSRetrievalHintPersistence interface {
	SaveSDSRetrievalHints(channelID, messageID string, envelopeHashes [][]byte) error
	LoadSDSRetrievalHints(channelID, messageID string) ([][]byte, error)
	SaveSDSMessageReceived(channelID, messageID string) error
	IsSDSMessageReceived(channelID, messageID string) (bool, error)
	DeleteExpiredSDSReceivedMessages() error
}
