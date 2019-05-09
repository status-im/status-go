package mailserver

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/status-im/whisper/whisperv6"
)

const (
	// DBKeyLength is a size of the envelope key.
	DBKeyLength  = common.HashLength + timestampLength + whisper.TopicLength
	CursorLength = common.HashLength + timestampLength
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

func (k *DBKey) Topic() whisper.TopicType {
	return whisper.BytesToTopic(k.raw[timestampLength+common.HashLength:])
}

func (k *DBKey) EnvelopeHash() common.Hash {
	return common.BytesToHash(k.raw[timestampLength : common.HashLength+timestampLength])
}

func (k *DBKey) Cursor() []byte {
	// We don't use the whole cursor for backward compatibility (also it's not needed)
	return k.raw[:CursorLength]
}

// NewDBKey creates a new DBKey with the given values.
func NewDBKey(timestamp uint32, topic whisper.TopicType, h common.Hash) *DBKey {
	var k DBKey
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint32(k.raw, timestamp)
	copy(k.raw[timestampLength:], h[:])
	copy(k.raw[timestampLength+common.HashLength:], topic[:])
	return &k
}
