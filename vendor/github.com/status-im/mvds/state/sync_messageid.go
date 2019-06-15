package state

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/status-im/mvds/protobuf"
)

type MessageID [32]byte
type GroupID [32]byte

// ID creates the MessageID for a Message
func ID(m protobuf.Message) MessageID {
	t := make([]byte, 8)
	binary.LittleEndian.PutUint64(t, uint64(m.Timestamp))

	b := append([]byte("MESSAGE_ID"), m.GroupId[:]...)
	b = append(b, t...)
	b = append(b, m.Body...)

	return sha256.Sum256(b)
}
