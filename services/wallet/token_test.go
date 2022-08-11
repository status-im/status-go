package wallet

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/sqlite"
)

func setupTestTokenDB(t *testing.T) (*TokenManager, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-token-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "wallet-token-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	return &TokenManager{db, nil, nil}, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestCustoms(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	rst, err := manager.getCustoms()
	require.NoError(t, err)
	require.Nil(t, rst)

	token := Token{
		Address:  common.Address{1},
		Name:     "Zilliqa",
		Symbol:   "ZIL",
		Decimals: 12,
		Color:    "#fa6565",
		ChainID:  777,
	}

	err = manager.upsertCustom(token)
	require.NoError(t, err)

	rst, err = manager.getCustoms()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, token, *rst[0])

	err = manager.deleteCustom(777, token.Address)
	require.NoError(t, err)

	rst, err = manager.getCustoms()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}
