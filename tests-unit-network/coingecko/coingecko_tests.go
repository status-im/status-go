package coingecko_tests

import (
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	mock_network "github.com/status-im/status-go/rpc/network/mock"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/market/coingecko"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"golang.org/x/exp/maps"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

func getGroupKeys(t *testing.T) []string {
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
	tm := token.NewTokenManager(walletDB, nil, nil, networkManager, appDB, nil, nil, nil, nil, nil)

	groupedTokens, err := tm.GetTokensGroupedByGroupKey()
	require.NoError(t, err)

	return maps.Keys(groupedTokens)
}

func TestFetchPrices(t *testing.T) {
	groupKeys := getGroupKeys(t)

	stdClient := coingecko.NewClient()
	_, err := stdClient.FetchPrices(groupKeys, []string{"USD"})
	require.NoError(t, err)
}

func TestFetchTokenMarketValues(t *testing.T) {
	groupKeys := getGroupKeys(t)

	stdClient := coingecko.NewClient()
	_, err := stdClient.FetchTokenMarketValues(groupKeys, "USD")
	require.NoError(t, err)
}
