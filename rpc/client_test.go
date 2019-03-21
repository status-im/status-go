package rpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

func TestBlockedRoutesCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"id": 1,
			"jsonrpc": "2.0",
			"result": "0x234234e22b9ffc2387e18636e0534534a3d0c56b0243567432453264c16e78a2adc"
		}`)
	}))
	defer ts.Close()

	gethRPCClient, err := gethrpc.Dial(ts.URL)
	require.NoError(t, err)

	c, err := NewClient(gethRPCClient, params.UpstreamRPCConfig{Enabled: false, URL: ""})
	require.NoError(t, err)

	for _, m := range blockedMethods {
		var (
			result interface{}
			err    error
		)

		err = c.Call(&result, m)
		require.EqualError(t, err, ErrMethodNotFound.Error())
		require.Nil(t, result)

		err = c.CallContext(context.Background(), &result, m)
		require.EqualError(t, err, ErrMethodNotFound.Error())
		require.Nil(t, result)

		err = c.CallContextIgnoringLocalHandlers(context.Background(), &result, m)
		require.EqualError(t, err, ErrMethodNotFound.Error())
		require.Nil(t, result)
	}
}

func TestBlockedRoutesRawCall(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"id": 1,
			"jsonrpc": "2.0",
			"result": "0x234234e22b9ffc2387e18636e0534534a3d0c56b0243567432453264c16e78a2adc"
		}`)
	}))
	defer ts.Close()

	gethRPCClient, err := gethrpc.Dial(ts.URL)
	require.NoError(t, err)

	c, err := NewClient(gethRPCClient, params.UpstreamRPCConfig{Enabled: false, URL: ""})
	require.NoError(t, err)

	for _, m := range blockedMethods {
		rawResult := c.CallRaw(fmt.Sprintf(`{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "%s",
			"params": ["0xc862bf3cf4565d46abcbadaf4712a8940bfea729a91b9b0e338eab5166341ab5"]
		}`, m))
		require.Contains(t, rawResult, fmt.Sprintf(`{"code":-32700,"message":"%s"}`, ErrMethodNotFound))
	}
}

func TestUpdateUpstreamURL(t *testing.T) {
	ts := createTestServer("")
	defer ts.Close()

	updatedUpstreamTs := createTestServer("")
	defer updatedUpstreamTs.Close()

	gethRPCClient, err := gethrpc.Dial(ts.URL)
	require.NoError(t, err)

	c, err := NewClient(gethRPCClient, params.UpstreamRPCConfig{Enabled: true, URL: ts.URL})
	require.NoError(t, err)
	require.Equal(t, ts.URL, c.upstreamURL)

	// cache the original upstream client
	originalUpstreamClient := c.upstream

	err = c.UpdateUpstreamURL(updatedUpstreamTs.URL)
	require.NoError(t, err)
	// the upstream cleint instance should change
	require.NotEqual(t, originalUpstreamClient, c.upstream)
	require.Equal(t, updatedUpstreamTs.URL, c.upstreamURL)
}

func createTestServer(resp string) *httptest.Server {
	if resp == "" {
		resp = `{
			"id": 1,
			"jsonrpc": "2.0",
			"result": "0x234234e22b9ffc2387e18636e0534534a3d0c56b0243567432453264c16e78a2adc"
		}`
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, resp)
	}))
}
