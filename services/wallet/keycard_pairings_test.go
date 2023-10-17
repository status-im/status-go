package wallet

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func TestKeycardPairingsFile(t *testing.T) {
	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDb, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	service := NewService(db, accountsDb, &rpc.Client{NetworkManager: network.NewManager(db)}, nil, nil, nil, nil, &params.NodeConfig{}, nil, nil, nil, nil)

	data, err := service.KeycardPairings().GetPairingsJSONFileContent()
	require.NoError(t, err)
	require.Equal(t, 0, len(data))

	pairingsFile, err := ioutil.TempFile("", "keycard-pairings.json")
	require.NoError(t, err)
	defer pairingsFile.Close()

	service.KeycardPairings().SetKeycardPairingsFile(pairingsFile.Name())

	dataToStore := []byte(`
	{"2b907a26ee4319ab50d7eda44b525f6a":{"key":"cc9d96f9b65b551595f3cf7c531beacda24b4937cece7fef70f5236ee80a0808","index":0},
	"4abcc337a3dfc7e89785c427ef32983b":{"key":"3543288f50b2c0bbb2745ffd7107bc3acd105197b97384342fe864e7391a7af7","index":3},
	"4b2e0fe09f997d7ce20320c971ad54df":{"key":"843edb10045d329f4ecfac73fe66f13deb7b2b685dd54a4b2d2d700d19062391","index":0},
	"7ce8e7456eb9025a97f3579490246cae":{"key":"b12a89ca66288f4239a2b58c2bb533df2694b613eb73fc55b72391497627766f","index":1}}
	`)

	err = service.KeycardPairings().SetPairingsJSONFileContent(dataToStore)
	require.NoError(t, err)

	data, err = service.KeycardPairings().GetPairingsJSONFileContent()
	require.NoError(t, err)
	require.Equal(t, len(dataToStore), len(data))
}
