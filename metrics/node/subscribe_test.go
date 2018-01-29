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
	// TODO
	// start and cancel using event
}
