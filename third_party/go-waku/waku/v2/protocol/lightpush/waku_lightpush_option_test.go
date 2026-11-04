package lightpush

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestLightPushOption(t *testing.T) {
	port, err := tests.FindFreePort(t, "", 5)
	require.NoError(t, err)

	host, err := tests.MakeHost(context.Background(), port, rand.Reader)
	require.NoError(t, err)

	options := []RequestOption{
		WithPeer("QmWLxGxG65CZ7vRj5oNXCJvbY9WkF9d9FxuJg8cg8Y7q3"),
		WithAutomaticPeerSelection(),
		WithFastestPeerSelection(),
		WithRequestID([]byte("requestID")),
		WithAutomaticRequestID(),
	}

	params := new(lightPushRequestParameters)
	params.host = host
	params.log = utils.Logger()

	for _, opt := range options {
		err := opt(params)
		require.NoError(t, err)
	}

	require.Equal(t, host, params.host)
	require.NotEqual(t, 0, len(params.selectedPeers))
	require.NotNil(t, params.requestID)

	maddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/12345/p2p/16Uiu2HAm8KUwGRruseAaEGD6xGg6XKrDo8Py5dwDoL9wUpCxawGy")
	require.NoError(t, err)

	options = []RequestOption{
		WithPeer("16Uiu2HAm8KUwGRruseAaEGD6xGg6XKrDo8Py5dwDoL9wUpCxawGy"),
		WithPeerAddr(maddr),
	}

	for idx, opt := range options {
		err = opt(params)
		if idx == 0 {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
