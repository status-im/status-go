package wallet

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/stretchr/testify/require"

	gomock "go.uber.org/mock/gomock"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	mock_reader "github.com/status-im/status-go/services/wallet/mock/reader"
	"github.com/status-im/status-go/services/wallet/onramp"
	mock_onramp "github.com/status-im/status-go/services/wallet/onramp/mock"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletconnect"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

// TestAPI_GetWalletConnectActiveSessions tames coverage
func TestAPI_GetWalletConnectActiveSessions(t *testing.T) {
	db, close := walletconnect.SetupTestDB(t)
	defer close()
	api := &API{
		s: &Service{db: db},
	}

	sessions, err := api.GetWalletConnectActiveSessions(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 0, len(sessions))
}

// TestAPI_HashMessageEIP191
func TestAPI_HashMessageEIP191(t *testing.T) {
	api := &API{}

	res := api.HashMessageEIP191(context.Background(), []byte("test"))
	require.Equal(t, "0x4a5c5d454721bbbb25540c3317521e71c373ae36458f960d2ad46ef088110e95", res.String())
}

func TestAPI_IsChecksumValidForAddress(t *testing.T) {
	api := &API{}

	res, err := api.IsChecksumValidForAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)
	require.False(t, res)

	res, err = api.IsChecksumValidForAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa")
	require.NoError(t, err)
	require.True(t, res)
}

func TestAPI_GetCryptoOnRamps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider0 := mock_onramp.NewMockProvider(ctrl)
	id0 := "provider0"
	provider0.EXPECT().ID().Return(id0).AnyTimes()
	provider1 := mock_onramp.NewMockProvider(ctrl)
	id1 := "provider1"
	provider1.EXPECT().ID().Return(id1).AnyTimes()
	providers := []onramp.Provider{provider0, provider1}
	onrampManager := onramp.NewManager(providers)

	api := &API{
		s: &Service{cryptoOnRampManager: onrampManager},
	}

	ctx := context.Background()

	// Check returned providers
	provider0.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id0}, nil)
	provider1.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id1}, nil)

	retProviders, err := api.GetCryptoOnRamps(ctx)
	require.NoError(t, err)
	require.Equal(t, len(providers), len(retProviders))
	require.Equal(t, id0, retProviders[0].ID)
	require.Equal(t, id1, retProviders[1].ID)

	// Check error handling
	provider0.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{}, errors.New("error"))
	provider1.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id1}, nil)
	retProviders, err = api.GetCryptoOnRamps(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(retProviders))
	require.Equal(t, id1, retProviders[0].ID)

	// Check URL retrieval
	provider1.EXPECT().GetURL(ctx, onramp.Parameters{}).Return("url", nil)
	url, err := api.GetCryptoOnRampURL(ctx, id1, onramp.Parameters{})
	require.NoError(t, err)
	require.Equal(t, "url", url)
}

func TestAPI_GetAddressDetails(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	defer appDB.Close()

	accountsDb, err := accounts.NewDB(appDB)
	require.NoError(t, err)
	defer accountsDb.Close()

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	defer db.Close()

	accountFeed := &event.Feed{}

	chainID := uint64(1)
	address := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	providerConfig := params.ProviderConfig{
		Enabled:  true,
		Name:     rpc.ProviderStatusProxy,
		User:     "user1",
		Password: "pass1",
	}
	providerConfigs := []params.ProviderConfig{providerConfig}

	// Create a new server that delays the response by 1 second
	serverWith1SecDelay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		fmt.Fprintln(w, `{"result": "0x10"}`)
	}))
	defer serverWith1SecDelay.Close()

	networks := []params.Network{
		{
			ChainID:       chainID,
			DefaultRPCURL: serverWith1SecDelay.URL + "/nodefleet/",
		},
	}
	config := rpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: chainID,
		Networks:        networks,
		DB:              appDB,
		WalletFeed:      nil,
		ProviderConfigs: providerConfigs,
	}
	c, err := rpc.NewClient(config)
	require.NoError(t, err)

	chainClient, err := c.EthClient(chainID)
	require.NoError(t, err)
	chainClient.SetWalletNotifier(func(chainID uint64, message string) {})
	c.SetWalletNotifier(func(chainID uint64, message string) {})

	service := NewService(db, accountsDb, appDB, c, accountFeed, nil, nil, nil, &params.NodeConfig{}, nil, nil, nil, nil, "")

	api := &API{
		s: service,
	}

	// Test getting address details using `GetAddressDetails` call, that always waits for the request to finish
	details, err := api.GetAddressDetails(context.Background(), 1, address)
	require.NoError(t, err)
	require.Equal(t, true, details.HasActivity)

	// empty params
	details, err = api.AddressDetails(context.Background(), &requests.AddressDetails{})
	require.Error(t, err)
	require.ErrorIs(t, err, requests.ErrAddresInvalid)
	require.Nil(t, details)

	// no response longer than the set timeout
	details, err = api.AddressDetails(context.Background(), &requests.AddressDetails{
		Address:               address,
		TimeoutInMilliseconds: 500,
	})
	require.NoError(t, err)
	require.Equal(t, false, details.HasActivity)

	// timeout longer than the response time
	details, err = api.AddressDetails(context.Background(), &requests.AddressDetails{
		Address:               address,
		TimeoutInMilliseconds: 1200,
	})
	require.NoError(t, err)
	require.Equal(t, true, details.HasActivity)

	// specific chain and timeout longer than the response time
	details, err = api.AddressDetails(context.Background(), &requests.AddressDetails{
		Address:               address,
		ChainIDs:              []uint64{chainID},
		TimeoutInMilliseconds: 1200,
	})
	require.NoError(t, err)
	require.Equal(t, true, details.HasActivity)
}

// TestAPI_FetchOrGetCachedWalletBalances
func TestAPI_FetchOrGetCachedWalletBalances(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	defer appDB.Close()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockReader := mock_reader.NewMockReaderInterface(mockCtrl)
	providerConfig := params.ProviderConfig{
		Enabled:  true,
		Name:     rpc.ProviderStatusProxy,
		User:     "user1",
		Password: "pass1",
	}
	providerConfigs := []params.ProviderConfig{providerConfig}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"result": "0x10"}`)
	}))
	defer server.Close()

	chainID := uint64(1)

	networks := []params.Network{
		{
			ChainID:       chainID,
			DefaultRPCURL: server.URL + "/nodefleet/",
		},
	}
	config := rpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: chainID,
		Networks:        networks,
		DB:              appDB,
		WalletFeed:      nil,
		ProviderConfigs: providerConfigs,
	}
	c, err := rpc.NewClient(config)
	require.NoError(t, err)

	testTokenAddress1 := common.Address{0x34}
	testAccAddress1 := common.Address{0x12}
	storageToken := token.StorageToken{
		Token: token.Token{
			Name:     "USD Tether",
			Symbol:   "USDT",
			Decimals: 18,
		},
		BalancesPerChain: map[uint64]token.ChainBalance{
			1: {
				RawBalance: "1000000000000000000",
				Balance:    nil,
				Address:    testAccAddress1,
				ChainID:    1,
				HasError:   false,
			},
		},
	}
	expectedTokens := map[common.Address][]token.StorageToken{
		testAccAddress1: {storageToken},
	}

	mockReader.EXPECT().FetchOrGetCachedWalletBalances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedTokens, nil)

	service := &Service{
		rpcClient: c,
	}

	api := &API{
		s:      service,
		reader: mockReader,
	}

	forceRefresh := true
	balances, err := api.FetchOrGetCachedWalletBalances(context.Background(), []common.Address{testTokenAddress1}, forceRefresh)
	require.NoError(t, err)
	require.NotNil(t, balances)
}
