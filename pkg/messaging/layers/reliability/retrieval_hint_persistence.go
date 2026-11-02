package reliability

// SDSRetrievalHintPersistence stores transport hashes associated with SDS
// message IDs. The SDS history itself bounds the lifetime of these records.
type SDSRetrievalHintPersistence interface {
	SaveSDSRetrievalHints(channelID, messageID string, envelopeHashes [][]byte) error
	LoadSDSRetrievalHints(channelID, messageID string) ([][]byte, error)
}
