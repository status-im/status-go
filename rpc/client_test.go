package rpc

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

func setupTestNetworkDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "rpc-network-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func TestBlockedRoutesCall(t *testing.T) {
	db, close := setupTestNetworkDB(t)
	defer close()

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

	c, err := NewClient(gethRPCClient, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db, nil)
	require.NoError(t, err)

	for _, m := range blockedMethods {
		var (
			result interface{}
			err    error
		)

		err = c.Call(&result, 1, m)
		require.EqualError(t, err, ErrMethodNotFound.Error())
		require.Nil(t, result)

		err = c.CallContext(context.Background(), &result, 1, m)
		require.EqualError(t, err, ErrMethodNotFound.Error())
		require.Nil(t, result)

		err = c.CallContextIgnoringLocalHandlers(context.Background(), &result, 1, m)
		require.EqualError(t, err, ErrMethodNotFound.Error())
		require.Nil(t, result)
	}
}

func TestBlockedRoutesRawCall(t *testing.T) {
	db, close := setupTestNetworkDB(t)
	defer close()

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

	c, err := NewClient(gethRPCClient, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db, nil)
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
	db, close := setupTestNetworkDB(t)
	defer close()

	ts := createTestServer("")
	defer ts.Close()

	updatedUpstreamTs := createTestServer("")
	defer updatedUpstreamTs.Close()

	gethRPCClient, err := gethrpc.Dial(ts.URL)
	require.NoError(t, err)

	c, err := NewClient(gethRPCClient, 1, params.UpstreamRPCConfig{Enabled: true, URL: ts.URL}, []params.Network{}, db, nil)
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

func TestGetClientsUsingCache(t *testing.T) {
	db, close := setupTestNetworkDB(t)
	defer close()

	providerConfig := params.ProviderConfig{
		Enabled:  true,
		Name:     ProviderStatusProxy,
		User:     "user1",
		Password: "pass1",
	}
	providerConfigs := []params.ProviderConfig{providerConfig}

	var wg sync.WaitGroup
	wg.Add(2) // 2 providers

	// Create a new ServeMux
	mux := http.NewServeMux()

	path1 := "/foo"
	path2 := "/bar"
	// Register handlers for different URL paths
	mux.HandleFunc(path1, func(w http.ResponseWriter, r *http.Request) {
		authToken := base64.StdEncoding.EncodeToString([]byte(providerConfig.User + ":" + providerConfig.Password))
		require.Equal(t, fmt.Sprintf("Basic %s", authToken), r.Header.Get("Authorization"))
		wg.Done()
	})

	mux.HandleFunc(path2, func(w http.ResponseWriter, r *http.Request) {
		authToken := base64.StdEncoding.EncodeToString([]byte(providerConfig.User + ":" + providerConfig.Password))
		require.Equal(t, fmt.Sprintf("Basic %s", authToken), r.Header.Get("Authorization"))
		wg.Done()
	})

	// Create a new server with the mux as the handler
	server := httptest.NewServer(mux)
	defer server.Close()

	networks := []params.Network{
		{
			ChainID:            1,
			DefaultRPCURL:      server.URL + path1,
			DefaultFallbackURL: server.URL + path2,
		},
	}
	c, err := NewClient(nil, 1, params.UpstreamRPCConfig{}, networks, db, providerConfigs)
	require.NoError(t, err)

	// Networks from DB must pick up DefaultRPCURL and DefaultFallbackURL
	chainClient, err := c.getClientUsingCache(networks[0].ChainID)
	require.NoError(t, err)
	require.NotNil(t, chainClient)

	// Make any call to provider. If test finishes, then all handlers were called and asserts inside them passed
	balance, err := chainClient.BalanceAt(context.TODO(), common.Address{0x1}, big.NewInt(1))
	assert.Error(t, err) // EOF, we dont return anything from the server, because of error iterate over all providers
	assert.Nil(t, balance)
	wg.Wait()
}
