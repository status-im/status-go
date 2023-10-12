package encryption

import (
	"encoding/binary"
	"time"

	"github.com/status-im/status-go/eth-node/crypto"
)

const keyBumpValue = uint64(10)

// GetCurrentTime64 returns the current unix time in milliseconds
func GetCurrentTime() uint64 {
	return (uint64)(time.Now().UnixNano() / int64(time.Millisecond))
}

// bumpKeyID takes a timestampID and returns its value incremented by the keyBumpValue
func bumpKeyID(timestampID uint64) uint64 {
	return timestampID + keyBumpValue
}

func generateHashRatchetKeyID(groupID []byte, timestamp uint64, keyBytes []byte) []byte {
	var keyMaterial []byte

	keyMaterial = append(keyMaterial, groupID...)

	timestampBytes := make([]byte, 8) // 8 bytes for a uint64
	binary.LittleEndian.PutUint64(timestampBytes, timestamp)
	keyMaterial = append(keyMaterial, timestampBytes...)

	keyMaterial = append(keyMaterial, keyBytes...)

	return crypto.Keccak256(keyMaterial)
}
