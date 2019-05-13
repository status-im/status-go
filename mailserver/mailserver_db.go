package mailserver

import (
	whisper "github.com/status-im/whisper/whisperv6"
	"time"
)

// DB is an interface to abstract interactions with the db so that the mailserver
// is agnostic to the underlaying technology used
type DB interface {
	Close() error
	// SaveEnvelope stores an envelope
	SaveEnvelope(*whisper.Envelope) error
	// GetEnvelope returns an rlp encoded envelope from the datastore
	GetEnvelope(*DBKey) ([]byte, error)
	// Prune removes envelopes older than time
	Prune(time.Time, int) (int, error)
	// BuildIterator returns an iterator over envelopes
	BuildIterator(query CursorQuery) (Iterator, error)
}

type Iterator interface {
	Next() bool
	DBKey() (*DBKey, error)
	Release()
	Error() error
	GetEnvelope(bloom []byte) ([]byte, error)
}

type CursorQuery struct {
	start  []byte
	end    []byte
	cursor []byte
	limit  uint32
	bloom  []byte
}
