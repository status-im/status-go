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
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/db/appdatabase"
	healthmanager "github.com/status-im/status-go/internal/healthmanager"
	"github.com/status-im/status-go/internal/healthmanager/rpcstatus"
	"github.com/status-im/status-go/internal/rpc/chain/rpclimiter"
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
		"/eth-rpc.status.im/nodefleet/foo",
		"/eth-rpc.status.im/infura/bar",
		"/eth-rpc.status.im/infura.io/baz",
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
				Type:         params.EmbeddedEthRpcProxyProviderType,
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
		params.EmbeddedEthRpcProxyProviderType,
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

func TestGetProviderRPCLimiterCircuitName(t *testing.T) {
	const host = "test.eth-rpc.status.im"

	newProvider := func(chainID uint64, rpsLimiter bool) params.RpcProvider {
		return params.RpcProvider{
			Name:             "smart-proxy",
			ChainID:          chainID,
			URL:              security.NewSensitiveString(fmt.Sprintf("https://%s/ethereum/mainnet/", host)),
			Type:             params.EmbeddedEthRpcProxyProviderType,
			AuthType:         params.NoAuth,
			EnableRPSLimiter: rpsLimiter,
			Enabled:          true,
		}
	}

	c := &Client{limiterPerProvider: make(map[string]*rpclimiter.RPCRpsLimiter)}

	// Circuit name must be non-empty even when the RPS limiter is disabled, so the
	// circuit breaker can short-circuit a consistently failing provider.
	limiter, circuitChain1, err := c.getProviderRPCLimiter(newProvider(1, false))
	require.NoError(t, err)
	require.Nil(t, limiter)
	require.Equal(t, fmt.Sprintf("%s-%d", host, 1), circuitChain1)

	// Same host but a different chain must yield a different circuit name, so a provider
	// that fails for one chain does not open the circuit for other chains sharing the host.
	limiter, circuitChain747474, err := c.getProviderRPCLimiter(newProvider(747474, false))
	require.NoError(t, err)
	require.Nil(t, limiter)
	require.Equal(t, fmt.Sprintf("%s-%d", host, 747474), circuitChain747474)
	require.NotEqual(t, circuitChain1, circuitChain747474)

	// With the RPS limiter enabled, the circuit name is still isolated per chain+host,
	// while the limiter itself is shared per host across chains.
	limiterChain1, circuitChain1Rps, err := c.getProviderRPCLimiter(newProvider(1, true))
	require.NoError(t, err)
	require.NotNil(t, limiterChain1)
	require.Equal(t, fmt.Sprintf("%s-%d", host, 1), circuitChain1Rps)

	limiterChain2, circuitChain2Rps, err := c.getProviderRPCLimiter(newProvider(2, true))
	require.NoError(t, err)
	require.NotNil(t, limiterChain2)
	require.Equal(t, fmt.Sprintf("%s-%d", host, 2), circuitChain2Rps)
	require.NotEqual(t, circuitChain1Rps, circuitChain2Rps)

	// Limiter is shared per host regardless of chain.
	require.Same(t, limiterChain1, limiterChain2)
}

func TestSetPausedPropagatesToProvidersHealthManager(t *testing.T) {
	c := &Client{
		healthMgr: healthmanager.NewBlockchainHealthManager(),
	}
	defer c.healthMgr.Stop()

	phm := healthmanager.NewProvidersHealthManager(1)
	phm.SetDownDebounce(50 * time.Millisecond)
	require.NoError(t, c.healthMgr.RegisterProvidersHealthManager(context.Background(), phm))

	ch := phm.Subscribe()
	defer phm.Unsubscribe(ch)

	phm.Update(context.Background(), []rpcstatus.RpcProviderCallStatus{
		{Name: "provider1", Timestamp: time.Now(), Err: nil},
	})
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial Up notification")
	}

	c.SetPaused(true)
	phm.Update(context.Background(), []rpcstatus.RpcProviderCallStatus{
		{Name: "provider1", Timestamp: time.Now(), Err: fmt.Errorf("down")},
	})
	select {
	case <-ch:
		t.Fatal("unexpected notification while client is paused")
	case <-time.After(120 * time.Millisecond):
	}

	c.SetPaused(false)
	select {
	case <-ch:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for debounced Down notification after resume")
	}
}
