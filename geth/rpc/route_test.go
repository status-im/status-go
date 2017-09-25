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
		require.True(t, router.routeRemote(method), "method "+method+" should routed to remote")
	}

	for _, method := range localMethods {
		t.Run(method, func(t *testing.T) {
			require.False(t, router.routeRemote(method), "method "+method+" should routed to local")
		})
	}
}

func TestRouteWithoutUpstream(t *testing.T) {
	router := newRouter(false)

	for _, method := range remoteMethods {
		require.False(t, router.routeRemote(method), "method "+method+" should routed to locally without UpstreamEnabled")
	}

	for _, method := range localMethods {
		require.False(t, router.routeRemote(method), "method "+method+" should routed to local")
	}
}
