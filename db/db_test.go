package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBKey(t *testing.T) {
	data1 := []byte{0x01, 0x02, 0x03}
	data2 := []byte{0x04, 0x05, 0x06, 0x07, 0x08}

	key := Key(PeersCache, data1, data2)
	assert.Equal(t, len(data1)+len(data2)+1, len(key))
	assert.Equal(t, byte(PeersCache), key[0])

	expectedKey := append([]byte{byte(PeersCache)}, data1...)
	expectedKey = append(expectedKey, data2...)

	assert.Equal(t, expectedKey, key)

	key = Key(DeduplicatorCache, data1)
	assert.Equal(t, len(data1)+1, len(key))
	assert.Equal(t, byte(DeduplicatorCache), key[0])

	expectedKey = append([]byte{byte(DeduplicatorCache)}, data1...)

	assert.Equal(t, expectedKey, key)

	key = Key(DeduplicatorCache, data2)
	assert.Equal(t, len(data2)+1, len(key))
	assert.Equal(t, byte(DeduplicatorCache), key[0])

	expectedKey = append([]byte{byte(DeduplicatorCache)}, data2...)

	assert.Equal(t, expectedKey, key)
}
