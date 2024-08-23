package ext

import (
	"testing"

	"github.com/status-im/status-go/eth-node/types"

	"github.com/stretchr/testify/assert"
)

func TestTopicsToBloom(t *testing.T) {
	t1 := stringToTopic("t1")
	b1 := types.TopicToBloom(t1)
	t2 := stringToTopic("t2")
	b2 := types.TopicToBloom(t2)
	t3 := stringToTopic("t3")
	b3 := types.TopicToBloom(t3)

	reqBloom := topicsToBloom(t1)
	assert.True(t, types.BloomFilterMatch(reqBloom, b1))
	assert.False(t, types.BloomFilterMatch(reqBloom, b2))
	assert.False(t, types.BloomFilterMatch(reqBloom, b3))

	reqBloom = topicsToBloom(t1, t2)
	assert.True(t, types.BloomFilterMatch(reqBloom, b1))
	assert.True(t, types.BloomFilterMatch(reqBloom, b2))
	assert.False(t, types.BloomFilterMatch(reqBloom, b3))

	reqBloom = topicsToBloom(t1, t2, t3)
	assert.True(t, types.BloomFilterMatch(reqBloom, b1))
	assert.True(t, types.BloomFilterMatch(reqBloom, b2))
	assert.True(t, types.BloomFilterMatch(reqBloom, b3))
}

func stringToTopic(s string) types.TopicType {
	return types.BytesToTopic([]byte(s))
}
