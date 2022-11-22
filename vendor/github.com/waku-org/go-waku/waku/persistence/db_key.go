package persistence

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

const (
	TimestampLength   = 8
	HashLength        = 32
	DigestLength      = HashLength
	PubsubTopicLength = HashLength
	DBKeyLength       = TimestampLength + PubsubTopicLength + DigestLength
)

type Hash [HashLength]byte

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

func (k *DBKey) Digest() []byte {
	return k.raw[TimestampLength+PubsubTopicLength : TimestampLength+PubsubTopicLength+DigestLength]
}

// NewDBKey creates a new DBKey with the given values.
func NewDBKey(timestamp uint64, pubsubTopic string, digest []byte) *DBKey {
	pubSubHash := sha256.Sum256([]byte(pubsubTopic))

	var k DBKey
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint64(k.raw, timestamp)
	copy(k.raw[TimestampLength:], pubSubHash[:])
	copy(k.raw[TimestampLength+PubsubTopicLength:], digest)

	return &k
}
