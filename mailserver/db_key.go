package mailserver

import (
	"encoding/binary"
	"errors"

	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
)

const (
	// DBKeyLength is a size of the envelope key.
	DBKeyLength  = statusproto.HashLength + timestampLength + whispertypes.TopicLength
	CursorLength = statusproto.HashLength + timestampLength
)

var (
	// ErrInvalidByteSize is returned when DBKey can't be created
	// from a byte slice because it has invalid length.
	ErrInvalidByteSize = errors.New("byte slice has invalid length")
)

// DBKey key to be stored in a db.
type DBKey struct {
	raw []byte
}

// Bytes returns a bytes representation of the DBKey.
func (k *DBKey) Bytes() []byte {
	return k.raw
}

func (k *DBKey) Topic() whispertypes.TopicType {
	return whispertypes.BytesToTopic(k.raw[timestampLength+statusproto.HashLength:])
}

func (k *DBKey) EnvelopeHash() statusproto.Hash {
	return statusproto.BytesToHash(k.raw[timestampLength : statusproto.HashLength+timestampLength])
}

func (k *DBKey) Cursor() []byte {
	// We don't use the whole cursor for backward compatibility (also it's not needed)
	return k.raw[:CursorLength]
}

// NewDBKey creates a new DBKey with the given values.
func NewDBKey(timestamp uint32, topic whispertypes.TopicType, h statusproto.Hash) *DBKey {
	var k DBKey
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint32(k.raw, timestamp)
	copy(k.raw[timestampLength:], h[:])
	copy(k.raw[timestampLength+statusproto.HashLength:], topic[:])
	return &k
}
