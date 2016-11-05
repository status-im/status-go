package main

import "C"
import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth"
)

const (
	testDataDir         = "../../.ethereumtest"
	testNodeSyncSeconds = 120
	testAddress         = "0x89b50b2b26947ccad43accaef76c21d175ad85f4"
	testAddressPassword = "asdf"
	newAccountPassword  = "badpassword"
	testAddress1        = "0xf82da7547534045b4e00442bc89e16186cf8c272"
)

func testExportedAPI(t *testing.T, done chan struct{}) {
	<-startTestNode(t)

	tests := []struct {
		name string
		fn   func(t *testing.T) bool
	}{
		{
			"test complete multiple queued transactions",
			testCompleteMultipleQueuedTransactions,
		},
		{
			"test discard multiple queued transactions",
			testDiscardMultipleQueuedTransactions,
		},
	}

	for _, test := range tests {
		if ok := test.fn(t); !ok {
			break
		}
	}

	done <- struct{}{}
}

func testCompleteMultipleQueuedTransactions(t *testing.T) bool {
	// obtain reference to status backend
	lightEthereum, err := geth.GetNodeManager().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return false
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIds := make(chan string, testTxCount)
	allTestTxCompleted := make(chan struct{}, 1)

	// replace transaction notification handler
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txId string
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txId = event["id"].(string)
			t.Logf("transaction queued (will be completed in a single call, once aggregated): {id: %s}\n", txId)

			txIds <- txId
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
			From:  geth.FromAddress(testAddress),
			To:    geth.ToAddress(testAddress1),
			Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
		})
		if err != nil {
			t.Errorf("unexpected error thrown: %v", err)
			return
		}

		if reflect.DeepEqual(txHashCheck, common.Hash{}) {
			t.Error("transaction returned empty hash")
			return
		}
	}

	// wait for transactions, and complete them in a single call
	completeTxs := func(txIdStrings string) {
		var parsedIds []string
		json.Unmarshal([]byte(txIdStrings), &parsedIds)

		parsedIds = append(parsedIds, "invalid-tx-id")
		updatedTxIdStrings, _ := json.Marshal(parsedIds)

		// complete
		resultsString := CompleteTransactions(C.CString(string(updatedTxIdStrings)), C.CString(testAddressPassword))
		resultsStruct := geth.CompleteTransactionsResult{}
		json.Unmarshal([]byte(C.GoString(resultsString)), &resultsStruct)
		results := resultsStruct.Results

		if len(results) != (testTxCount+1) || results["invalid-tx-id"].Error != "transaction hash not found" {
			t.Errorf("cannot complete txs: %v", results)
			return
		}
		for txId, txResult := range results {
			if txId != txResult.Id {
				t.Errorf("tx id not set in result: expected id is %s", txId)
				return
			}
			if txResult.Error != "" && txId != "invalid-tx-id" {
				t.Errorf("invalid error for %s", txId)
				return
			}
			if txResult.Hash == "0x0000000000000000000000000000000000000000000000000000000000000000" && txId != "invalid-tx-id" {
				t.Errorf("invalid hash (expected non empty hash): %s", txId)
				return
			}

			if txResult.Hash != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txResult.Hash)
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txId := range parsedIds {
			if backend.TransactionQueue().Has(status.QueuedTxId(txId)) {
				t.Errorf("txqueue should not have test tx at this point (it should be completed): %s", txId)
				return
			}
		}
	}
	go func() {
		var txIdStrings []string
		for i := 0; i < testTxCount; i++ {
			txIdStrings = append(txIdStrings, <-txIds)
		}

		txIdJSON, _ := json.Marshal(txIdStrings)
		completeTxs(string(txIdJSON))
		allTestTxCompleted <- struct{}{}
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		go sendTx()
	}

	select {
	case <-allTestTxCompleted:
	// pass
	case <-time.After(20 * time.Second):
		t.Error("test timed out")
		return false
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	return true
}

func testDiscardMultipleQueuedTransactions(t *testing.T) bool {
	// obtain reference to status backend
	lightEthereum, err := geth.GetNodeManager().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return false
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIds := make(chan string, testTxCount)
	allTestTxDiscarded := make(chan struct{}, 1)

	// replace transaction notification handler
	txFailedEventCallCount := 0
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txId string
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txId = event["id"].(string)
			t.Logf("transaction queued (will be discarded soon): {id: %s}\n", txId)

			if !backend.TransactionQueue().Has(status.QueuedTxId(txId)) {
				t.Errorf("txqueue should still have test tx: %s", txId)
				return
			}

			txIds <- txId
		}

		if envelope.Type == geth.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction return event received: {id: %s}\n", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := status.ErrQueuedTxDiscarded.Error()
			if receivedErrMessage != expectedErrMessage {
				t.Errorf("unexpected error message received: got %v", receivedErrMessage)
				return
			}

			receivedErrCode := event["error_code"].(string)
			if receivedErrCode != geth.SendTransactionDiscardedErrorCode {
				t.Errorf("unexpected error code received: got %v", receivedErrCode)
				return
			}

			txFailedEventCallCount++
			if txFailedEventCallCount == testTxCount {
				allTestTxDiscarded <- struct{}{}
			}
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
			From:  geth.FromAddress(testAddress),
			To:    geth.ToAddress(testAddress1),
			Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
		})
		if err != status.ErrQueuedTxDiscarded {
			t.Errorf("expected error not thrown: %v", err)
			return
		}

		if !reflect.DeepEqual(txHashCheck, common.Hash{}) {
			t.Error("transaction returned hash, while it shouldn't")
			return
		}
	}

	// wait for transactions, and discard immediately
	discardTxs := func(txIdStrings string) {
		var parsedIds []string
		json.Unmarshal([]byte(txIdStrings), &parsedIds)

		parsedIds = append(parsedIds, "invalid-tx-id")
		updatedTxIdStrings, _ := json.Marshal(parsedIds)

		// discard
		discardResultsString := DiscardTransactions(C.CString(string(updatedTxIdStrings)))
		discardResultsStruct := geth.DiscardTransactionsResult{}
		json.Unmarshal([]byte(C.GoString(discardResultsString)), &discardResultsStruct)
		discardResults := discardResultsStruct.Results

		if len(discardResults) != 1 || discardResults["invalid-tx-id"].Error != "transaction hash not found" {
			t.Errorf("cannot discard txs: %v", discardResults)
			return
		}

		// try completing discarded transaction
		completeResultsString := CompleteTransactions(C.CString(string(updatedTxIdStrings)), C.CString(testAddressPassword))
		completeResultsStruct := geth.CompleteTransactionsResult{}
		json.Unmarshal([]byte(C.GoString(completeResultsString)), &completeResultsStruct)
		completeResults := completeResultsStruct.Results

		if len(completeResults) != (testTxCount + 1) {
			t.Error("unexpected number of errors (call to CompleteTransaction should not succeed)")
		}
		for txId, txResult := range completeResults {
			if txId != txResult.Id {
				t.Errorf("tx id not set in result: expected id is %s", txId)
				return
			}
			if txResult.Error != "transaction hash not found" {
				t.Errorf("invalid error for %s", txResult.Hash)
				return
			}
			if txResult.Hash != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				t.Errorf("invalid hash (expected zero): %s", txResult.Hash)
				return
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txId := range parsedIds {
			if backend.TransactionQueue().Has(status.QueuedTxId(txId)) {
				t.Errorf("txqueue should not have test tx at this point (it should be discarded): %s", txId)
				return
			}
		}
	}
	go func() {
		var txIdStrings []string
		for i := 0; i < testTxCount; i++ {
			txIdStrings = append(txIdStrings, <-txIds)
		}

		txIdJSON, _ := json.Marshal(txIdStrings)
		discardTxs(string(txIdJSON))
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		go sendTx()
	}

	select {
	case <-allTestTxDiscarded:
	// pass
	case <-time.After(20 * time.Second):
		t.Error("test timed out")
		return false
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	return true
}

func startTestNode(t *testing.T) <-chan struct{} {
	syncRequired := false
	if _, err := os.Stat(filepath.Join(testDataDir, "testnet")); os.IsNotExist(err) {
		syncRequired = true
	}

	waitForNodeStart := make(chan struct{}, 1)
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		t.Log(jsonEvent)
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
		}
		if envelope.Type == geth.EventNodeStarted {
			if syncRequired {
				t.Logf("Sync is required, it will take %d seconds", testNodeSyncSeconds)
				time.Sleep(testNodeSyncSeconds * time.Second) // LES syncs headers, so that we are up do date when it is done
			} else {
				time.Sleep(5 * time.Second)
			}

			// now we can proceed with tests
			waitForNodeStart <- struct{}{}
		}
	})

	response := StartNode(C.CString(testDataDir))
	err := geth.JSONError{}

	json.Unmarshal([]byte(C.GoString(response)), &err)
	if err.Error != "" {
		t.Error("cannot start node")
	}

	return waitForNodeStart
}
