package accountsstore

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name())
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{Name: "string", Address: common.Address{0xff}}
	require.NoError(t, db.SaveAccount(expected))
	accounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, expected, accounts[0])
}

func TestConfig(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	expected := Account{Name: "string", Address: common.Address{0xff}}
	require.NoError(t, db.SaveAccount(expected))
	conf := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}
	require.NoError(t, db.SaveConfig(expected.Address, "node-config", conf))
	var rst params.NodeConfig
	require.NoError(t, db.GetConfig(expected.Address, "node-config", &rst))
	require.Equal(t, conf, rst)
}
