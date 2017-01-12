package geth_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/status-im/status-go/geth"
)

const (
	testAddress         = "0xadaf150b905cf5e6a778e553e15a139b6618bbb7"
	testAddressPassword = "asdfasdf"
	newAccountPassword  = "badpassword"
	testAddress1        = "0xadd4d1d02e71c7360c53296968e59d57fd15e2ba"

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
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			panic(fmt.Errorf("cannot unmarshal event's JSON: %s", jsonEvent))
		}
		if envelope.Type == geth.EventNodeCrashed {
			geth.TriggerDefaultNodeNotificationHandler(jsonEvent)
			return
		}

		if jsonEvent == `{"type":"node.started","event":{}}` {
			signalRecieved <- struct{}{}
		}
	})

	err := geth.PrepareTestNode()
	if err != nil {
		panic(err)
	}

	<-signalRecieved // block and wait for either panic or successful signal
	abortPanic <- struct{}{}

	os.Exit(m.Run())
}
