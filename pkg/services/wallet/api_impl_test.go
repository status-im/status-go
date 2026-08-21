package wallet

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/networks"
	network_mock "github.com/status-im/status-go/pkg/services/networks/mock"
	"github.com/status-im/status-go/pkg/services/networks/testutil"
	"github.com/status-im/status-go/pkg/services/wallet/requests"
	"github.com/status-im/status-go/pkg/services/wallet/token"
	mock_tokenbalances "github.com/status-im/status-go/pkg/services/wallet/tokenbalances/mock"
)

func TestAPI_GetAddressDetails(t *testing.T) {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	defer appDB.Close()

	accountsDb, err := accounts.NewDB(appDB)
	require.NoError(t, err)
	defer accountsDb.Close()

	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	defer db.Close()

	accountsPublisher := pubsub.NewPublisher()

	chainID := uint64(1)
	address := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	// Create a new server that delays the response by 1 second
	serverWith1SecDelay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		fmt.Fprintln(w, `{"result": "0x10"}`)
	}))
	defer serverWith1SecDelay.Close()

	testNetworks := []params.Network{
		*testutil.CreateNetwork(chainID, "Ethereum Mainnet", []params.RpcProvider{
			*params.NewEthRpcProxyProvider(chainID, "Test Provider", security.NewSensitiveString(serverWith1SecDelay.URL+"/nodefleet/"), false, false),
		},
		),
	}

	testNetworks = networkhelper.OverrideBasicAuth(testNetworks, params.EmbeddedEthRpcProxyProviderType, true, security.NewSensitiveString(gofakeit.Username()), security.NewSensitiveString(gofakeit.LetterN(5)))
	require.NotEmpty(t, testNetworks)

	networkManager := networks.NewManager(appDB, nil)
	require.NotNil(t, networkManager)
	require.NoError(t, networkManager.InitEmbeddedNetworks(testNetworks))

	config := rpc.ClientConfig{
		NetworkManager: networkManager,
	}
	c, err := rpc.NewClient(config)
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)
	mockNetworkManager := network_mock.NewMockManagerInterface(mockCtrl)
	mockNetworkManager.EXPECT().GetActiveNetworks().DoAndReturn(func() ([]*params.Network, error) {
		active := make([]*params.Network, 0, len(testNetworks))
		for i := range testNetworks {
			active = append(active, &testNetworks[i])
		}
		return active, nil
	}).AnyTimes()
	mockNetworkManager.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	mockNetworkManager.EXPECT().GetTestNetworksEnabled().Return(false, nil).AnyTimes()

	tokenManager, err := token.NewTokenManager(db, c, nil, mockNetworkManager, appDB, nil, nil, nil, accountsDb, 0, 0)
	require.NoError(t, err)

	service, err := NewService(db, accountsDb, appDB, c, accountsPublisher, nil, nil, &params.NodeConfig{}, nil, nil, nil, nil, tokenManager)
	require.NoError(t, err)

	tokenbalancesFetcher := mock_tokenbalances.NewMockFetcherIface(mockCtrl)

	service.tokenBalancesFetcher = tokenbalancesFetcher

	api := NewAPI(service)

	tokenbalancesFetcher.EXPECT().FetchSingle(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, chainID uint64, tokenAddress common.Address, accountAddress common.Address) (*big.Int, error) {
		// Delay the response by 1 second
		timer := time.NewTimer(1 * time.Second)
		select {
		case <-timer.C:
			return big.NewInt(1000000000000000000), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}).AnyTimes()

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
