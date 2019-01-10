package mailserver

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// DBKeyLength is a size of the envelope key.
	DBKeyLength = common.HashLength + timestampLength
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
func NewDBKey(t uint32, h common.Hash) *DBKey {
	var k DBKey
	k.timestamp = t
	k.hash = h
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint32(k.raw, k.timestamp)
	copy(k.raw[4:], k.hash[:])
	return &k
}

// NewDBKeyFromBytes creates a DBKey from a byte slice.
func NewDBKeyFromBytes(b []byte) *DBKey {
	return &DBKey{
		raw:       b,
		timestamp: binary.BigEndian.Uint32(b),
		hash:      common.BytesToHash(b[4:]),
	}
}
