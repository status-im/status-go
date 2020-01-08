package mailserver

import (
	"errors"
	"time"
)

const (
	maxMessagesRequestPayloadLimit = 1000
)

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

func (r *MessagesRequestPayload) SetDefaults() {
	if r.Limit == 0 {
		r.Limit = maxQueryLimit
	}

	if r.Upper == 0 {
		r.Upper = uint32(time.Now().Unix() + whisperTTLSafeThreshold)
	}
}

func (r MessagesRequestPayload) Validate() error {
	if r.Upper < r.Lower {
		return errors.New("query range is invalid: lower > upper")
	}
	if r.Upper-r.Lower > uint32(maxQueryRange.Seconds()) {
		return errors.New("query range must be smaller or equal to 24 hours")
	}
	if len(r.Bloom) == 0 {
		return errors.New("bloom filter is empty")
	}
	if r.Limit > maxMessagesRequestPayloadLimit {
		return errors.New("limit exceeds the maximum allowed value")
	}
	return nil
}
