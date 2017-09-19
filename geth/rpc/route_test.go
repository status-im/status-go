package rpc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// some of the upstream examples
var localMethods = []string{"some_weirdo_method", "shh_newMessageFilter", "net_version"}

func TestRouteWithUpstream(t *testing.T) {
	router := newRouter(true)

	for _, method := range remoteMethods {
		require.True(t, router.routeRemote(method))
	}

	for _, method := range localMethods {
		t.Run(method, func(t *testing.T) {
			require.False(t, router.routeRemote(method))
		})
	}
}

func TestRouteWithoutUpstream(t *testing.T) {
	router := newRouter(false)

	for _, method := range remoteMethods {
		require.True(t, router.routeRemote(method))
	}

	for _, method := range localMethods {
		require.True(t, router.routeRemote(method))
	}
}
