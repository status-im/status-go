package mailserver

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// DBKeyLength is a size of the envelope key.
	DBKeyLength = common.HashLength + timestampLength
)

var (
	// ErrInvalidByteSize is returned when DBKey can't be created
	// from a byte slice because it has invalid length.
	ErrInvalidByteSize = errors.New("byte slice has invalid length")
)

// DBKey key to be stored in a db.
type DBKey struct {
	timestamp uint32
	hash      common.Hash
	raw       []byte
}

// Bytes returns a bytes representation of the DBKey.
func (k *DBKey) Bytes() []byte {
	return k.raw
}

// NewDBKey creates a new DBKey with the given values.
func NewDBKey(timestamp uint32, h common.Hash) *DBKey {
	var k DBKey
	k.timestamp = timestamp
	k.hash = h
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint32(k.raw, k.timestamp)
	copy(k.raw[4:], k.hash[:])
	return &k
}

// NewDBKeyFromBytes creates a DBKey from a byte slice.
func NewDBKeyFromBytes(b []byte) (*DBKey, error) {
	if len(b) != DBKeyLength {
		return nil, ErrInvalidByteSize
	}

	return &DBKey{
		raw:       b,
		timestamp: binary.BigEndian.Uint32(b),
		hash:      common.BytesToHash(b[4:]),
	}, nil
}

// mustNewDBKeyFromBytes panics if creating a key from a byte slice fails.
// Check if a byte slice has DBKeyLength length before using it.
func mustNewDBKeyFromBytes(b []byte) *DBKey {
	k, err := NewDBKeyFromBytes(b)
	if err != nil {
		panic(err)
	}
	return k
}
