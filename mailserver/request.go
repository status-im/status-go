package mailserver

// MessagesRequestPayload is a payload sent to the Mail Server.
type MessagesRequestPayload struct {
	// Lower is a lower bound of time range for which messages are requested.
	Lower uint32
	// Upper is a lower bound of time range for which messages are requested.
	Upper uint32
	// Bloom is a bloom filter to filter envelopes.
	Bloom []byte
	// Limit is the max number of envelopes to return.
	Limit uint32
	// Cursor is used for pagination of the results.
	Cursor []byte
	// Batch set to true indicates that the client supports batched response.
	Batch bool
}
