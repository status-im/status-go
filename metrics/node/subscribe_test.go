package node

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
)

func TestSubscribeServerEventsWithoutServer(t *testing.T) {
	node, err := node.New(&node.Config{})
	require.NoError(t, err)
	require.EqualError(t, SubscribeServerEvents(context.TODO(), node), "server is unavailable")
}

func TestSubscribeServerEvents(t *testing.T) {
	node, err := node.New(&node.Config{})
	require.NoError(t, err)
	err = node.Start()
	require.NoError(t, err)
	defer node.Stop() //nolint: errcheck

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		err := SubscribeServerEvents(ctx, node)
		require.NoError(t, err)
		close(done)
	}()

	cancel()
	<-done
}
