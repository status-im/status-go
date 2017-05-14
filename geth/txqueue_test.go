package geth_test

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/params"
)

func TestQueuedContracts(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// create an account
	sampleAddress, _, _, err := geth.CreateAccount(testConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}

	if err = geth.Logout(); err != nil {
		t.Fatal(err)
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(5*time.Second, completeQueuedTransaction, "TestQueuedContracts")

	// replace transaction notification handler
	var txHash = common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint :dupl
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != status.ErrInvalidCompleteTxSender {
				t.Errorf("expected error on queued transaction[%v] not thrown: expected %v, got %v", event["id"], status.ErrInvalidCompleteTxSender, err)
				return
			}

			// the second call will also fail (we are logged in as different user)
			if err := geth.SelectAccount(sampleAddress, testConfig.Account1.Password); err != nil {
				t.Errorf("cannot select account: %v", sampleAddress)
				return
			}
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != status.ErrInvalidCompleteTxSender {
				t.Errorf("expected error on queued transaction[%v] not thrown: expected %v, got %v", event["id"], status.ErrInvalidCompleteTxSender, err)
				return
			}

			// the third call will work as expected (as we are logged in with correct credentials)
			if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
				t.Errorf("cannot select account: %v", testConfig.Account1.Address)
				return
			}
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != nil {
				t.Errorf("cannot complete queued transaction[%v]: %v", event["id"], err)
				return
			}

			t.Logf("contract transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			close(completeQueuedTransaction)
		}
	})

	//  this call blocks, up until Complete Transaction is called
	byteCode, err := hexutil.Decode(`0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029`)
	if err != nil {
		t.Error(err)
		return
	}
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From: geth.FromAddress(testConfig.Account1.Address),
		To:   nil, // marker, contract creation is expected
		//Value: (*hexutil.Big)(new(big.Int).Mul(big.NewInt(1), common.Ether)),
		Gas:  (*hexutil.Big)(big.NewInt(params.DefaultGas)),
		Data: byteCode,
	})
	if err != nil {
		t.Errorf("Test failed: cannot send transaction: %v", err)
	}

	if txHashCheck.Hex() != txHash.Hex() {
		t.Errorf("Transaction hash returned from SendTransaction is invalid: expected %s, got %s", txHashCheck.Hex(), txHash.Hex())
		return
	}

	<-completeQueuedTransaction

	if reflect.DeepEqual(txHashCheck, common.Hash{}) {
		t.Error("Test failed: transaction was never queued or completed")
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return
	}
}

func TestQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// create an account
	sampleAddress, _, _, err := geth.CreateAccount(testConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}

	if err = geth.Logout(); err != nil {
		t.Fatal(err)
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(5*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txHash = common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint: dupl
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != status.ErrInvalidCompleteTxSender {
				t.Errorf("expected error on queued transaction[%v] not thrown: expected %v, got %v", event["id"], status.ErrInvalidCompleteTxSender, err)
				return
			}

			// the second call will also fail (we are logged in as different user)
			if err := geth.SelectAccount(sampleAddress, testConfig.Account1.Password); err != nil {
				t.Errorf("cannot select account: %v", sampleAddress)
				return
			}
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != status.ErrInvalidCompleteTxSender {
				t.Errorf("expected error on queued transaction[%v] not thrown: expected %v, got %v", event["id"], status.ErrInvalidCompleteTxSender, err)
				return
			}

			// the third call will work as expected (as we are logged in with correct credentials)
			if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
				t.Errorf("cannot select account: %v", testConfig.Account1.Address)
				return
			}
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != nil {
				t.Errorf("cannot complete queued transaction[%v]: %v", event["id"], err)
				return
			}

			t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			close(completeQueuedTransaction)
		}
	})

	//  this call blocks, up until Complete Transaction is called
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  geth.FromAddress(testConfig.Account1.Address),
		To:    geth.ToAddress(testConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("Test failed: cannot send transaction: %v", err)
	}

	if txHashCheck.Hex() != txHash.Hex() {
		t.Errorf("Transaction hash returned from SendTransaction is invalid: expected %s, got %s", txHashCheck.Hex(), txHash.Hex())
		return
	}

	<-completeQueuedTransaction

	if reflect.DeepEqual(txHashCheck, common.Hash{}) {
		t.Error("Test failed: transaction was never queued or completed")
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return
	}
}

func TestDoubleCompleteQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", testConfig.Account1.Address)
		return
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txID string
	txFailedEventCalled := false
	txHash := common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be failed and completed on the second call): {id: %s}\n", txID)

			// try with wrong password
			// make sure that tx is NOT removed from the queue (by re-trying with the correct password)
			if _, err = geth.CompleteTransaction(txID, testConfig.Account1.Password+"wrong"); err != keystore.ErrDecrypt {
				t.Errorf("expects wrong password error, but call succeeded (or got another error: %v)", err)
				return
			}

			if txCount := backend.TransactionQueue().Count(); txCount != 1 {
				t.Errorf("txqueue cannot be empty, as tx has failed: expected = 1, got = %d", txCount)
				return
			}

			// now try to complete transaction, but with the correct password
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != nil {
				t.Errorf("cannot complete queued transaction[%v]: %v", event["id"], err)
				return
			}

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			if txCount := backend.TransactionQueue().Count(); txCount != 0 {
				t.Errorf("txqueue must be empty, as tx has completed: expected = 0, got = %d", txCount)
				return
			}

			t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
		}

		if envelope.Type == geth.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction return event received: {id: %s}\n", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := "could not decrypt key with given passphrase"
			if receivedErrMessage != expectedErrMessage {
				t.Errorf("unexpected error message received: got %v", receivedErrMessage)
				return
			}

			receivedErrCode := event["error_code"].(string)
			if receivedErrCode != geth.SendTransactionPasswordErrorCode {
				t.Errorf("unexpected error code received: got %v", receivedErrCode)
				return
			}

			txFailedEventCalled = true
		}
	})

	//  this call blocks, and should return on *second* attempt to CompleteTransaction (w/ the correct password)
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  geth.FromAddress(testConfig.Account1.Address),
		To:    geth.ToAddress(testConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("cannot send transaction: %v", err)
		return
	}

	if txHashCheck.Hex() != txHash.Hex() {
		t.Errorf("tx hash returned from SendTransaction is invalid: expected %s, got %s", txHashCheck.Hex(), txHash.Hex())
		return
	}

	if reflect.DeepEqual(txHashCheck, common.Hash{}) {
		t.Error("transaction was never queued or completed")
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return
	}

	if !txFailedEventCalled {
		t.Error("expected tx failure signal is not received")
		return
	}
}

func TestDiscardQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", testConfig.Account1.Address)
		return
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestDiscardQueuedTransactions")

	// replace transaction notification handler
	var txID string
	txFailedEventCalled := false
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be discarded soon): {id: %s}\n", txID)

			if !backend.TransactionQueue().Has(status.QueuedTxID(txID)) {
				t.Errorf("txqueue should still have test tx: %s", txID)
				return
			}

			// discard
			err := geth.DiscardTransaction(txID)
			if err != nil {
				t.Errorf("cannot discard tx: %v", err)
				return
			}

			// try completing discarded transaction
			_, err = geth.CompleteTransaction(txID, testConfig.Account1.Password)
			if err.Error() != "transaction hash not found" {
				t.Error("expects tx not found, but call to CompleteTransaction succeeded")
				return
			}

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			if backend.TransactionQueue().Has(status.QueuedTxID(txID)) {
				t.Errorf("txqueue should not have test tx at this point (it should be discarded): %s", txID)
				return
			}

			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
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

			txFailedEventCalled = true
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() is called
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  geth.FromAddress(testConfig.Account1.Address),
		To:    geth.ToAddress(testConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != status.ErrQueuedTxDiscarded {
		t.Errorf("expected error not thrown: %v", err)
		return
	}

	if !reflect.DeepEqual(txHashCheck, common.Hash{}) {
		t.Error("transaction returned hash, while it shouldn't")
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return
	}

	if !txFailedEventCalled {
		t.Error("expected tx failure signal is not received")
		return
	}
}

func TestCompleteMultipleQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", testConfig.Account1.Address)
		return
	}

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIDs := make(chan string, testTxCount)
	allTestTxCompleted := make(chan struct{}, 1)

	// replace transaction notification handler
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txID string
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be completed in a single call, once aggregated): {id: %s}\n", txID)

			txIDs <- txID
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
			From:  geth.FromAddress(testConfig.Account1.Address),
			To:    geth.ToAddress(testConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
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
	completeTxs := func(txIDStrings string) {
		var parsedIDs []string
		if err := json.Unmarshal([]byte(txIDStrings), &parsedIDs); err != nil {
			t.Error(err)
			return
		}

		parsedIDs = append(parsedIDs, "invalid-tx-id")
		updatedTxIDStrings, _ := json.Marshal(parsedIDs)

		// complete
		results := geth.CompleteTransactions(string(updatedTxIDStrings), testConfig.Account1.Password)
		if len(results) != (testTxCount+1) || results["invalid-tx-id"].Error.Error() != "transaction hash not found" {
			t.Errorf("cannot complete txs: %v", results)
			return
		}
		for txID, txResult := range results {
			if txResult.Error != nil && txID != "invalid-tx-id" {
				t.Errorf("invalid error for %s", txID)
				return
			}
			if txResult.Hash.Hex() == "0x0000000000000000000000000000000000000000000000000000000000000000" && txID != "invalid-tx-id" {
				t.Errorf("invalid hash (expected non empty hash): %s", txID)
				return
			}

			if txResult.Hash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txResult.Hash.Hex())
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txID := range parsedIDs {
			if backend.TransactionQueue().Has(status.QueuedTxID(txID)) {
				t.Errorf("txqueue should not have test tx at this point (it should be completed): %s", txID)
				return
			}
		}
	}
	go func() {
		var txIDStrings []string
		for i := 0; i < testTxCount; i++ {
			txIDStrings = append(txIDStrings, <-txIDs)
		}

		txIDJSON, _ := json.Marshal(txIDStrings)
		completeTxs(string(txIDJSON))
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
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return
	}
}

func TestDiscardMultipleQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", testConfig.Account1.Address)
		return
	}

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIDs := make(chan string, testTxCount)
	allTestTxDiscarded := make(chan struct{}, 1)

	// replace transaction notification handler
	txFailedEventCallCount := 0
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txID string
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be discarded soon): {id: %s}\n", txID)

			if !backend.TransactionQueue().Has(status.QueuedTxID(txID)) {
				t.Errorf("txqueue should still have test tx: %s", txID)
				return
			}

			txIDs <- txID
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
			From:  geth.FromAddress(testConfig.Account1.Address),
			To:    geth.ToAddress(testConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
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
	discardTxs := func(txIDStrings string) {
		var parsedIDs []string
		if err := json.Unmarshal([]byte(txIDStrings), &parsedIDs); err != nil {
			t.Error(err)
			return
		}

		parsedIDs = append(parsedIDs, "invalid-tx-id")
		updatedTxIDStrings, _ := json.Marshal(parsedIDs)

		// discard
		discardResults := geth.DiscardTransactions(string(updatedTxIDStrings))
		if len(discardResults) != 1 || discardResults["invalid-tx-id"].Error.Error() != "transaction hash not found" {
			t.Errorf("cannot discard txs: %v", discardResults)
			return
		}

		// try completing discarded transaction
		completeResults := geth.CompleteTransactions(string(updatedTxIDStrings), testConfig.Account1.Password)
		if len(completeResults) != (testTxCount + 1) {
			t.Error("unexpected number of errors (call to CompleteTransaction should not succeed)")
		}
		for _, txResult := range completeResults {
			if txResult.Error.Error() != "transaction hash not found" {
				t.Errorf("invalid error for %s", txResult.Hash.Hex())
				return
			}
			if txResult.Hash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				t.Errorf("invalid hash (expected zero): %s", txResult.Hash.Hex())
				return
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txID := range parsedIDs {
			if backend.TransactionQueue().Has(status.QueuedTxID(txID)) {
				t.Errorf("txqueue should not have test tx at this point (it should be discarded): %s", txID)
				return
			}
		}
	}
	go func() {
		var txIDStrings []string
		for i := 0; i < testTxCount; i++ {
			txIDStrings = append(txIDStrings, <-txIDs)
		}

		txIDJSON, _ := json.Marshal(txIDStrings)
		discardTxs(string(txIDJSON))
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
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return
	}
}

func TestNonExistentQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", testConfig.Account1.Address)
		return
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	geth.SetDefaultNodeNotificationHandler(func(string) {})

	// try completing non-existing transaction
	if _, err = geth.CompleteTransaction("some-bad-transaction-id", testConfig.Account1.Password); err == nil {
		t.Error("error expected and not received")
		return
	}
	if err != status.ErrQueuedTxIDNotFound {
		t.Errorf("unexpected error received: expected '%s', got: '%s'", status.ErrQueuedTxIDNotFound.Error(), err.Error())
		return
	}
}

func TestEvictionOfQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", testConfig.Account1.Address)
		return
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txHash = common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("Transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			// next call is the very same one, but with the correct password
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != nil {
				t.Errorf("cannot complete queued transaction[%v]: %v", event["id"], err)
				return
			}

			t.Logf("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
		}
	})

	txQueue := backend.TransactionQueue()
	var i = 0
	txIDs := [status.DefaultTxQueueCap + 5 + 10]status.QueuedTxID{}
	backend.SetTransactionQueueHandler(func(queuedTx status.QueuedTx) {
		t.Logf("%d. Transaction queued (queue size: %d): {id: %v}\n", i, txQueue.Count(), queuedTx.ID)
		txIDs[i] = queuedTx.ID
		i++
	})

	if txQueue.Count() != 0 {
		t.Errorf("transaction count should be zero: %d", txQueue.Count())
		return
	}

	for i := 0; i < 10; i++ {
		go backend.SendTransaction(nil, status.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(3 * time.Second)

	t.Logf("Number of transactions queued: %d. Queue size (shouldn't be more than %d): %d", i, status.DefaultTxQueueCap, txQueue.Count())

	if txQueue.Count() != 10 {
		t.Errorf("transaction count should be 10: got %d", txQueue.Count())
		return
	}

	for i := 0; i < status.DefaultTxQueueCap+5; i++ { // stress test by hitting with lots of goroutines
		go backend.SendTransaction(nil, status.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(5 * time.Second)

	if txQueue.Count() > status.DefaultTxQueueCap {
		t.Errorf("transaction count should be %d (or %d): got %d", status.DefaultTxQueueCap, status.DefaultTxQueueCap-1, txQueue.Count())
		return
	}

	for _, txID := range txIDs {
		txQueue.Remove(txID)
	}

	if txQueue.Count() != 0 {
		t.Errorf("transaction count should be zero: %d", txQueue.Count())
		return
	}
}
