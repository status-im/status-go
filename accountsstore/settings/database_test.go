package settings

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "settings-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name(), "settings-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestConfig(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	conf := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}
	require.NoError(t, db.SaveConfig("node-config", conf))
	var rst params.NodeConfig
	require.NoError(t, db.GetConfig("node-config", &rst))
	require.Equal(t, conf, rst)
}

func TestBlob(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	tag := "random-param"
	param := 10
	require.NoError(t, db.SaveConfig(tag, param))
	expected, err := json.Marshal(param)
	require.NoError(t, err)
	rst, err := db.GetBlob(tag)
	require.NoError(t, err)
	require.Equal(t, expected, rst)
}

func TestSaveSubAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	accounts := []Account{
		{Address: common.Address{0x01}, Chat: true, Wallet: true},
		{Address: common.Address{0x02}},
	}
	require.NoError(t, db.SaveAccounts(accounts))
}
