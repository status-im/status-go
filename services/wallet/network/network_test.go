package network

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
)

var initNetworks = []Network{
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
	},
	{
		ChainID:                3,
		ChainName:              "Ropsten",
		RPCURL:                 "https://ropsten.infura.io/nKmXgiFgc2KqtoQ8BCGJ",
		BlockExplorerURL:       "https://ropsten.etherscan.io/",
		IconURL:                "",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  1,
		Enabled:                false,
	},
	{
		ChainID:                4,
		ChainName:              "Rinkeby",
		RPCURL:                 "https://rinkeby.infura.io/nKmXgiFgc2KqtoQ8BCGJ",
		BlockExplorerURL:       "https://rinkeby.etherscan.io/",
		IconURL:                "",
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 true,
		Layer:                  1,
		Enabled:                false,
	},
	{
		ChainID:                5,
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
	},
}

func setupTestNetworkDB(t *testing.T) (*sql.DB, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-network-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "wallet-network-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestInitNetwork(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := &Manager{db: db}
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	network := nm.Find(1)
	require.NotNil(t, network)
	require.Equal(t, (uint64)(1), network.ChainID)
}

func TestGet(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := &Manager{db: db}
	err := nm.Init(initNetworks)
	require.NoError(t, err)

	networks, err := nm.Get(true)
	require.Nil(t, err)
	require.Equal(t, 2, len(networks))
}

func TestDelete(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := &Manager{db: db}
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

	nm := &Manager{db: db}
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
