package geth_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/status-im/status-go/geth"
)

const (
	testAddress         = "0x89b50b2b26947ccad43accaef76c21d175ad85f4"
	testAddressPassword = "asdf"
	newAccountPassword  = "badpassword"

	whisperMessage1 = "test message 1 (K1 -> K1)"
	whisperMessage2 = "test message 2 (K1 -> '')"
	whisperMessage3 = "test message 3 ('' -> '')"
	whisperMessage4 = "test message 4 ('' -> K1)"
	whisperMessage5 = "test message 5 (K2 -> K1)"
)

func TestMain(m *testing.M) {
	syncRequired := false
	if _, err := os.Stat(filepath.Join(geth.TestDataDir, "testnet")); os.IsNotExist(err) {
		syncRequired = true
	}
	// make sure you panic if node start signal is not received
	signalRecieved := make(chan struct{}, 1)
	abortPanic := make(chan struct{}, 1)
	if syncRequired {
		geth.PanicAfter(geth.TestNodeSyncSeconds*time.Second, abortPanic, "TestNodeSetup")
	} else {
		geth.PanicAfter(10*time.Second, abortPanic, "TestNodeSetup")
	}

	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		if jsonEvent == `{"type":"node.started","event":{}}` {
			signalRecieved <- struct{}{}
		}
	})

	err := geth.PrepareTestNode()
	if err != nil {
		panic(err)
		return
	}

	<-signalRecieved // block and wait for either panic or successful signal
	abortPanic <- struct{}{}

	os.Exit(m.Run())
}
