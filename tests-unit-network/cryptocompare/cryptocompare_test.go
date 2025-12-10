package cryptocompare_tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/params"
	mock_network "github.com/status-im/status-go/rpc/network/mock"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/market/cryptocompare"
	"github.com/status-im/status-go/services/wallet/token"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/t/helpers"

	"go.uber.org/mock/gomock"
)

func getTokenSymbols(t *testing.T) []*tokentypes.Token {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	networksList := []params.Network{
		{
			ChainID: w_common.EthereumMainnet,
		},
		{
			ChainID: w_common.OptimismMainnet,
		},
		{
			ChainID: w_common.ArbitrumMainnet,
		},
		{
			ChainID: w_common.BaseMainnet,
		},
		{
			ChainID: w_common.BSCMainnet,
		},
	}

	ptrNetworkList := make([]*params.Network, 0, len(networksList))
	for i := range networksList {
		ptrNetworkList = append(ptrNetworkList, &networksList[i])
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkManager := mock_network.NewMockManagerInterface(ctrl)
	networkManager.EXPECT().Get(gomock.Any()).Return(ptrNetworkList, nil).AnyTimes()
	networkManager.EXPECT().GetAll().Return(ptrNetworkList, nil).AnyTimes()
	networkManager.EXPECT().GetEmbeddedNetworks().Return(networksList).AnyTimes()

	// Skeleton token store to get full list of tokens
	tm, err := token.NewTokenManager(walletDB, nil, nil, networkManager, appDB, nil, nil, nil, nil, time.Hour, time.Hour)
	require.NoError(t, err)

	err = tm.Start(context.Background())
	require.NoError(t, err)

	tokens, err := tm.GetTokensOfInterestForActiveNetworksMode()
	require.NoError(t, err)
	require.Greater(t, len(tokens), 0)

	return tokens
}

func TestFetchPrices(t *testing.T) {
	tokens := getTokenSymbols(t)

	stdClient := cryptocompare.NewClient()
	_, err := stdClient.FetchPrices(tokens, []string{"USD"})
	require.NoError(t, err)
}

func TestFetchTokenMarketValues(t *testing.T) {
	tokens := getTokenSymbols(t)

	stdClient := cryptocompare.NewClient()
	_, err := stdClient.FetchTokenMarketValues(tokens, "USD")
	require.NoError(t, err)
}
