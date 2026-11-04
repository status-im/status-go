package filter

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestFilterOption(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	// subscribe options
	options := []FilterSubscribeOption{
		WithPeer("QmWLxGxG65CZ7vRj5oNXCJvbY9WkF9d9FxuJg8cg8Y7q3"),
		WithAutomaticPeerSelection(),
		WithFastestPeerSelection(),
	}

	params := new(FilterSubscribeParameters)
	params.host = host
	params.log = utils.Logger()

	for _, opt := range options {
		err = opt(params)
		require.NoError(t, err)
	}

	require.Equal(t, host, params.host)
	require.NotEqual(t, 0, params.selectedPeers)

	// Unsubscribe options
	options2 := []FilterSubscribeOption{
		WithAutomaticRequestID(),
		UnsubscribeAll(),
		WithPeer("QmWLxGxG65CZ7vRj5oNXCJvbY9WkF9d9FxuJg8cg8Y7q3"),
	}

	params2 := new(FilterSubscribeParameters)

	for _, opt := range options2 {
		err := opt(params2)
		require.NoError(t, err)
	}

	require.NotEqual(t, 0, params2.selectedPeers)
	require.True(t, params2.unsubscribeAll)

	// Mutually Exclusive options
	maddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/12345/p2p/16Uiu2HAm8KUwGRruseAaEGD6xGg6XKrDo8Py5dwDoL9wUpCxawGy")
	require.NoError(t, err)
	options3 := []FilterSubscribeOption{
		WithPeer("16Uiu2HAm8KUwGRruseAaEGD6xGg6XKrDo8Py5dwDoL9wUpCxawGy"),
		WithPeerAddr(maddr),
	}

	params3 := new(FilterSubscribeParameters)

	for idx, opt := range options3 {
		err := opt(params3)
		if idx == 0 {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}

	require.NotEqual(t, 0, params2.selectedPeers)
	require.True(t, params2.unsubscribeAll)

}
