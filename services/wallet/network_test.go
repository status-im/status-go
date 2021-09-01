package wallet

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
)

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

	nm := &NetworkManager{db: db}
	err := nm.init()
	require.NoError(t, err)

	network := nm.find(1)
	require.NotNil(t, network)
	require.Equal(t, (uint64)(1), network.ChainID)
}

func TestGet(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := &NetworkManager{db: db}
	err := nm.init()
	require.NoError(t, err)

	networks, err := nm.get(true)
	require.Nil(t, err)
	require.Equal(t, 2, len(networks))
}

func TestDelete(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := &NetworkManager{db: db}
	err := nm.init()
	require.NoError(t, err)

	err = nm.delete(1)
	require.NoError(t, err)
	networks, err := nm.get(true)
	require.Nil(t, err)
	require.Equal(t, 1, len(networks))
}

func TestUpsert(t *testing.T) {
	db, stop := setupTestNetworkDB(t)
	defer stop()

	nm := &NetworkManager{db: db}
	err := nm.init()
	require.NoError(t, err)

	network := nm.find(1)
	require.NotNil(t, network)

	newName := "New Chain Name"
	network.ChainName = newName
	err = nm.upsert(network)
	require.Nil(t, err)

	network = nm.find(1)
	require.NotNil(t, network)
	require.Equal(t, newName, network.ChainName)
}
