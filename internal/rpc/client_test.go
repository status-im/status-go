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

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/security"
)

func setupTestNetworkDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "rpc-network-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
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
		Networks: networks,
		DB:       db,
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
}
