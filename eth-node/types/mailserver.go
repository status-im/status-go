package types

// MailServerResponse is the response payload sent by the mailserver.
type MailServerResponse struct {
	LastEnvelopeHash Hash
	Cursor           []byte
	Error            error
}

// SyncMailRequest contains details which envelopes should be synced
// between Mail Servers.
type SyncMailRequest struct {
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
}

// SyncEventResponse is a response from the Mail Server
// form which the peer received envelopes.
type SyncEventResponse struct {
	Cursor []byte
	Error  string
}
