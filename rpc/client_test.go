package rpc

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/internal/security"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
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

	config := ClientConfig{
		Client:          gethRPCClient,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
	}
	c, err := NewClient(config)
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

	config := ClientConfig{
		Client:          gethRPCClient,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
	}
	c, err := NewClient(config)
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
func TestGetClientsUsingCache(t *testing.T) {
	db, close := setupTestNetworkDB(t)
	defer close()

	var wg sync.WaitGroup
	wg.Add(3) // 3 providers

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Define paths for providers
	paths := []string{
		"/api.status.im/nodefleet/foo",
		"/api.status.im/infura/bar",
		"/api.status.im/infura.io/baz",
	}
	user, password := gofakeit.Username(), gofakeit.LetterN(5)

	authHandler := func(w http.ResponseWriter, r *http.Request) {
		authToken := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
		require.Equal(t, fmt.Sprintf("Basic %s", authToken), r.Header.Get("Authorization"))
		wg.Done()
	}

	// Register handlers for different URL paths
	for _, path := range paths {
		mux.HandleFunc(path, authHandler)
	}

	// Create a new server with the mux as the handler
	server := httptest.NewServer(mux)
	defer server.Close()

	// Functor to create providers
	createProviders := func(baseURL string, paths []string) []params.RpcProvider {
		var providers []params.RpcProvider
		for i, path := range paths {
			providers = append(providers, params.RpcProvider{
				Name:         fmt.Sprintf("Provider%d", i+1),
				ChainID:      1,
				URL:          security.NewSensitiveString(baseURL).Append(path),
				Type:         params.EmbeddedProxyProviderType,
				AuthType:     params.BasicAuth,
				AuthLogin:    security.NewSensitiveString("incorrectUser"),
				AuthPassword: security.NewSensitiveString("incorrectPwd"), // will be replaced by correct values by OverrideBasicAuth
				Enabled:      true,
			})
		}
		return providers
	}

	networks := []params.Network{
		{
			ChainID:      1,
			ChainName:    "foo",
			RpcProviders: createProviders(server.URL, paths), // Create providers dynamically
		},
	}

	networks = networkhelper.OverrideBasicAuth(
		networks,
		params.EmbeddedProxyProviderType,
		true,
		security.NewSensitiveString(user),
		security.NewSensitiveString(password))

	config := ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        networks,
		DB:              db,
		WalletFeed:      nil,
	}

	c, err := NewClient(config)
	require.NoError(t, err)

	// Networks from DB must pick up RpcProviders
	chainClient, err := c.getClientUsingCache(networks[0].ChainID)
	require.NoError(t, err)
	require.NotNil(t, chainClient)

	// Make any call to provider. If test finishes, then all handlers were called and asserts inside them passed
	balance, err := chainClient.BalanceAt(context.TODO(), common.Address{0x1}, big.NewInt(1))
	assert.Error(t, err) // EOF, we don't return anything from the server, causing iteration over all providers
	assert.Nil(t, balance)
	wg.Wait()
}

func TestUserAgent(t *testing.T) {
	require.True(t, strings.HasPrefix(rpcUserAgentName, "procuratee-desktop/"))
	require.True(t, strings.HasPrefix(rpcUserAgentUpstreamName, "procuratee-desktop-upstream/"))
}
