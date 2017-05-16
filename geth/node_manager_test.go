package geth_test

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/params"
)

var testConfig *geth.TestConfig

func TestMain(m *testing.M) {
	// load shared test configuration
	var err error
	testConfig, err = geth.LoadTestConfig()
	if err != nil {
		panic(err)
	}

	// run tests
	retCode := m.Run()

	//time.Sleep(25 * time.Second) // to give some time to propagate txs to the rest of the network
	os.Exit(retCode)
}

func TestResetChainData(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	if err := geth.NodeManagerInstance().ResetChainData(); err != nil {
		t.Error(err)
		return
	}

	// allow some time to re-sync
	time.Sleep(testConfig.Node.SyncSeconds * time.Second)

	// now make sure that everything is intact
	TestQueuedTransactions(t)
}

func TestNetworkChange(t *testing.T) {
	var firstBlock struct {
		Hash common.Hash `json:"hash"`
	}

	// Ropsten
	dataDir1, err := ioutil.TempDir(os.TempDir(), "ropsten")
	if err != nil {
		t.Fatal(err)
	}
	configJSON1 := `{
		"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
		"DataDir": "` + dataDir1 + `",
		"HTTPPort": 8845,
		"LogEnabled": true,
		"LogLevel": "INFO"
	}`

	// Rinkeby
	dataDir2, err := ioutil.TempDir(os.TempDir(), "rinkeby")
	if err != nil {
		t.Fatal(err)
	}
	configJSON2 := `{
		"NetworkId": ` + strconv.Itoa(params.RinkebyNetworkID) + `,
		"DataDir": "` + dataDir2 + `",
		"HTTPPort": 8845,
		"LogEnabled": true,
		"LogLevel": "INFO"
	}`

	// start Ropsten
	nodeConfig, err := params.LoadNodeConfig(configJSON1)
	if err != nil {
		t.Fatal(err)
	}
	if err := geth.CreateAndRunNode(nodeConfig); err != nil {
		t.Fatal(err)
	}

	// allow some time to sync
	time.Sleep(15 * time.Second)

	// validate the hash of the first block
	rpcClient, err := geth.NodeManagerInstance().Node().GethStack().Attach()
	if err != nil {
		t.Fatal(err)
	}
	if err := rpcClient.CallContext(context.Background(), &firstBlock, "eth_getBlockByNumber", "0x0", true); err != nil {
		t.Fatal(err)
	}
	expectedHash := "0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"
	if firstBlock.Hash.Hex() != expectedHash {
		t.Errorf("unexpected genesis hash, expected: %v, got: %v", expectedHash, firstBlock.Hash.Hex())
	}

	// stop running node
	if err := geth.NodeManagerInstance().StopNode(); err != nil {
		t.Fatal(err)
	}

	// start Rinkeby
	nodeConfig, err = params.LoadNodeConfig(configJSON2)
	if err != nil {
		t.Fatal(err)
	}
	if err := geth.CreateAndRunNode(nodeConfig); err != nil {
		t.Fatal(err)
	}

	// allow some time to sync
	time.Sleep(15 * time.Second)

	// validate the hash of the first block
	rpcClient, err = geth.NodeManagerInstance().Node().GethStack().Attach()
	if err != nil {
		t.Fatal(err)
	}
	if err := rpcClient.CallContext(context.Background(), &firstBlock, "eth_getBlockByNumber", "0x0", true); err != nil {
		t.Fatal(err)
	}
	expectedHash = "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177"
	if firstBlock.Hash.Hex() != expectedHash {
		t.Errorf("unexpected genesis hash, expected: %v, got: %v", expectedHash, firstBlock.Hash.Hex())
	}

	// stop running node
	if err := geth.NodeManagerInstance().StopNode(); err != nil {
		t.Fatal(err)
	}
}
