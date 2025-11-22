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

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network/testutil"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/token"
	mock_tokenbalances "github.com/status-im/status-go/services/wallet/tokenbalances/mock"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

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

	accountsPublisher := pubsub.NewPublisher()

	chainID := uint64(1)
	address := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	// Create a new server that delays the response by 1 second
	serverWith1SecDelay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		fmt.Fprintln(w, `{"result": "0x10"}`)
	}))
	defer serverWith1SecDelay.Close()

	networks := []params.Network{
		*testutil.CreateNetwork(chainID, "Ethereum Mainnet", []params.RpcProvider{
			*params.NewProxyProvider(chainID, "Test Provider", security.NewSensitiveString(serverWith1SecDelay.URL+"/nodefleet/"), false),
		},
		),
	}

	networks = networkhelper.OverrideBasicAuth(networks, params.EmbeddedProxyProviderType, true, security.NewSensitiveString(gofakeit.Username()), security.NewSensitiveString(gofakeit.LetterN(5)))
	require.NotEmpty(t, networks)

	config := rpc.ClientConfig{
		Networks: networks,
		DB:       appDB,
	}
	c, err := rpc.NewClient(config)
	require.NoError(t, err)

	tokenManager := token.NewTokenManager(db, c, nil, nil, appDB, nil, nil, nil, accountsDb, token.NewPersistence(db))

	service := NewService(db, accountsDb, c, accountsPublisher, nil, nil, &params.NodeConfig{}, nil, nil, nil, nil, tokenManager, "")

	mockCtrl := gomock.NewController(t)
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
