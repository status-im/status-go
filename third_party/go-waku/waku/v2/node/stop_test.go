package node

import (
	"context"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
)

// TestStopAfterFailedStart checks that Stop() is safe when Start() failed before
// the host was created (here the listen port is already in use).
func TestStopAfterFailedStart(t *testing.T) {
	// Occupy a TCP port so libp2p host creation fails to listen on it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)

	key, err := tests.RandomHex(32)
	require.NoError(t, err)
	prvKey, err := crypto.HexToECDSA(key)
	require.NoError(t, err)

	wakuNode, err := New(WithPrivateKey(prvKey), WithHostAddress(addr))
	require.NoError(t, err)

	err = wakuNode.Start(context.Background())
	require.Error(t, err, "Start should fail when the port is in use")

	require.NotPanics(t, func() { wakuNode.Stop() })
}
