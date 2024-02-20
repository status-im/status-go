package network

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/helpers"
)

var initNetworks = []params.Network{
	{
		ChainID:                1,
		ChainName:              "Ethereum Mainnet",
		RPCURL:                 "https://mainnet.infura.io/nKmXgiFgc2KqtoQ8BCGJ",
		BlockExplorerURL:       "https://etherscan.io/",
		IconURL:                "",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  1,
		Enabled:                true,
		RelatedChainID:         11155111,
	},
	{
		ChainID:                11155111,
		ChainName:              "Goerli",
		RPCURL:                 "http://goerli.blockscout.com/",
		BlockExplorerURL:       "https://goerli.etherscan.io/",
		IconURL:                "",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  1,
		Enabled:                false,
		RelatedChainID:         1,
	},
	{
		ChainID:                10,
		ChainName:              "Optimistic Ethereum",
		RPCURL:                 "https://mainnet.infura.io/nKmXgiFgc2KqtoQ8BCGJ",
		BlockExplorerURL:       "https://optimistic.etherscan.io",
		IconURL:                "",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  2,
		Enabled:                true,
		RelatedChainID:         420,
	},
}

func setupTestNetworkDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "wallet-network-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func TestInitNetwork(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := NewManager(db)
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	network := nm.Find(1)
	require.NotNil(t, network)
	require.Equal(t, (uint64)(1), network.ChainID)
}

func TestGet(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := NewManager(db)
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	networks, err := nm.Get(true)
	require.Nil(t, err)
	require.Equal(t, 2, len(networks))
}

func TestGetCombinedNetworks(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := NewManager(db)
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	combinedNetworks, err := nm.GetCombinedNetworks()
	require.Nil(t, err)
	require.Equal(t, 2, len(combinedNetworks))
	require.Equal(t, uint64(1), combinedNetworks[0].Prod.ChainID)
	require.Equal(t, uint64(11155111), combinedNetworks[0].Test.ChainID)
	require.Equal(t, uint64(10), combinedNetworks[1].Prod.ChainID)
	require.Nil(t, combinedNetworks[1].Test)
}

func TestDelete(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := NewManager(db)
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	err = nm.Delete(1)
	require.NoError(t, err)
	networks, err := nm.Get(true)
	require.Nil(t, err)
	require.Equal(t, 1, len(networks))
}

func TestUpsert(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := NewManager(db)
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	network := nm.Find(1)
	require.NotNil(t, network)

	newName := "New Chain Name"
	network.ChainName = newName
	err = nm.Upsert(network)
	require.Nil(t, err)

	network = nm.Find(1)
	require.NotNil(t, network)
	require.Equal(t, newName, network.ChainName)
}
