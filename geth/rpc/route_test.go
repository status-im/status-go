package rpc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// some of the upstream examples
var upstreamMethods = []string{"some_weirdo_method", "eth_syncing", "eth_getBalance", "eth_call", "eth_getTransactionReceipt"}

func TestRouteWithUpstream(t *testing.T) {
	router := newRouter(true)

	for _, method := range localMethods {
		require.True(t, router.routeLocally(method))
	}

	for _, method := range upstreamMethods {
		require.False(t, router.routeLocally(method))
	}
}

func TestRouteWithoutUpstream(t *testing.T) {
	router := newRouter(false)

	for _, method := range localMethods {
		require.True(t, router.routeLocally(method))
	}

	for _, method := range upstreamMethods {
		require.True(t, router.routeLocally(method))
	}
}
