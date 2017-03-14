package geth_test

import (
	"os"
	"testing"
	"time"

	"github.com/status-im/status-go/geth"
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
