package mailserver

import (
	"encoding/binary"
	"errors"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
)

const (
	// DBKeyLength is a size of the envelope key.
	DBKeyLength  = protocol.HashLength + timestampLength + whispertypes.TopicLength
	CursorLength = protocol.HashLength + timestampLength
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
	return whispertypes.BytesToTopic(k.raw[timestampLength+protocol.HashLength:])
}

func (k *DBKey) EnvelopeHash() protocol.Hash {
	return protocol.BytesToHash(k.raw[timestampLength : protocol.HashLength+timestampLength])
}

func (k *DBKey) Cursor() []byte {
	// We don't use the whole cursor for backward compatibility (also it's not needed)
	return k.raw[:CursorLength]
}

// NewDBKey creates a new DBKey with the given values.
func NewDBKey(timestamp uint32, topic whispertypes.TopicType, h protocol.Hash) *DBKey {
	var k DBKey
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint32(k.raw, timestamp)
	copy(k.raw[timestampLength:], h[:])
	copy(k.raw[timestampLength+protocol.HashLength:], topic[:])
	return &k
}
