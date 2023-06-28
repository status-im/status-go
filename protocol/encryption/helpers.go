package encryption

import (
	"time"
)

const keyBumpValue = uint32(10)

// GetCurrentTime returns the current unix time in milliseconds
func GetCurrentTime() uint32 {
	return (uint32)(time.Now().UnixNano() / int64(time.Millisecond))
}

// bumpKeyID takes a timestampID and returns its value incremented by the keyBumpValue
func bumpKeyID(timestampID uint32) uint32 {
	return timestampID + keyBumpValue
}
