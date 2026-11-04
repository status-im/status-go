package utils

import (
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"testing"
)

func TestMultiaddrAsKeyMap(t *testing.T) {
	addr := "/ip4/127.0.0.1/tcp/63634/p2p/16Uiu2HAmJSLz9nyDMmjRpZzHAxMFiEUeszWgPReekzAroCgkTgbD"
	m1 := make(map[multiaddr.Multiaddr]struct{})
	mm1, err := multiaddr.NewMultiaddr(addr)
	require.NoError(t, err)
	m1[mm1] = struct{}{}
	_, ok := m1[mm1]
	require.True(t, ok)
	mm2, err := multiaddr.NewMultiaddr(addr)
	require.NoError(t, err)
	_, ok = m1[mm2]
	require.False(t, ok)

	m2 := make(map[multiaddr.Multiaddr]struct{})
	m2[mm2] = struct{}{}
	require.False(t, maps.Equal(m1, m2))
}

func TestMultiAddrSetEquals(t *testing.T) {
	addr := "/ip4/127.0.0.1/tcp/63634/p2p/16Uiu2HAmJSLz9nyDMmjRpZzHAxMFiEUeszWgPReekzAroCgkTgbD"
	ma1, err := multiaddr.NewMultiaddr(addr)
	require.NoError(t, err)
	ma2, err := multiaddr.NewMultiaddr(addr)
	require.NoError(t, err)

	m1 := map[string]multiaddr.Multiaddr{
		addr: ma1,
	}
	m2 := map[string]multiaddr.Multiaddr{
		addr: ma2,
	}
	require.True(t, MultiAddrSetEquals(m1, m2))
}
