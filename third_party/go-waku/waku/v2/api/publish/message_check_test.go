package publish

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAddAndDelete(t *testing.T) {
	ctx := context.TODO()
	messageSentCheck := NewMessageSentCheck(ctx, nil, nil, nil, nil, nil, nil)

	messageSentCheck.Add("topic", [32]byte{1}, 1)
	messageSentCheck.Add("topic", [32]byte{2}, 2)
	messageSentCheck.Add("topic", [32]byte{3}, 3)
	messageSentCheck.Add("another-topic", [32]byte{4}, 4)

	require.Equal(t, uint32(1), messageSentCheck.messageIDs["topic"][[32]byte{1}])
	require.Equal(t, uint32(2), messageSentCheck.messageIDs["topic"][[32]byte{2}])
	require.Equal(t, uint32(3), messageSentCheck.messageIDs["topic"][[32]byte{3}])
	require.Equal(t, uint32(4), messageSentCheck.messageIDs["another-topic"][[32]byte{4}])

	messageSentCheck.DeleteByMessageIDs([]common.Hash{[32]byte{1}, [32]byte{2}})
	require.NotNil(t, messageSentCheck.messageIDs["topic"])
	require.Equal(t, uint32(3), messageSentCheck.messageIDs["topic"][[32]byte{3}])

	messageSentCheck.DeleteByMessageIDs([]common.Hash{[32]byte{3}})
	require.Nil(t, messageSentCheck.messageIDs["topic"])

	require.Equal(t, uint32(4), messageSentCheck.messageIDs["another-topic"][[32]byte{4}])
}
