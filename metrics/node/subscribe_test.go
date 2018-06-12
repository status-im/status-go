package node

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
)

func TestSubscribeServerEventsWithoutServer(t *testing.T) {
	gethNode, err := node.New(&node.Config{})
	require.NoError(t, err)
	require.EqualError(t, SubscribeServerEvents(context.TODO(), gethNode), "server is unavailable")
}

func TestSubscribeServerEvents(t *testing.T) {
	gethNode, err := node.New(&node.Config{})
	require.NoError(t, err)
	err = gethNode.Start()
	require.NoError(t, err)
	defer func() {
		err := gethNode.Stop()
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		err := SubscribeServerEvents(ctx, gethNode)
		require.NoError(t, err)
		close(done)
	}()

	cancel()
	<-done
}
