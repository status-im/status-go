package wakuext

import (
	"io/ioutil"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/walletdatabase"
)

func TestInitProtocol(t *testing.T) {
	config := params.NodeConfig{
		RootDataDir: t.TempDir(),
		ShhextConfig: params.ShhextConfig{
			InstallationID:          "2",
			PFSEnabled:              true,
			MailServerConfirmations: true,
			ConnectionTarget:        10,
		},
	}
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	waku := gethbridge.NewGethWakuWrapper(waku.New(nil, nil))
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	nodeWrapper := ext.NewTestNodeWrapper(nil, waku)
	service := New(config, nodeWrapper, nil, nil, db)

	appDB, cleanupDB, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "db.sql")
	defer func() { require.NoError(t, cleanupDB()) }()
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "multi-accounts-tests-")
	require.NoError(t, err)
	multiAccounts, err := multiaccounts.InitializeDB(tmpfile.Name())
	require.NoError(t, err)

	acc := &multiaccounts.Account{KeyUID: "0xdeadbeef"}

	walletDB, cleanupWalletDB, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "db-wallet.sql")
	defer func() { require.NoError(t, cleanupWalletDB()) }()
	require.NoError(t, err)

	err = service.InitProtocol("Test", privateKey, appDB, walletDB, nil, multiAccounts, acc, nil, nil, nil, nil, nil, zap.NewNop())
	require.NoError(t, err)
}
