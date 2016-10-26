package geth_test

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth"
)

func TestQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	accountManager, err := geth.GetNodeManager().AccountManager()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	// create an account
	address, _, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.GetNodeManager().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txHash = common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			if txHash, err = geth.CompleteTransaction(event["id"].(string), testAddressPassword); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
				return
			}

			t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
		}
	})

	// send from the same test account (which is guaranteed to have ether)
	from, err := utils.MakeAddress(accountManager, testAddress)
	if err != nil {
		t.Errorf("could not retrieve account from address: %v", err)
		return
	}

	to, err := utils.MakeAddress(accountManager, address)
	if err != nil {
		t.Errorf("could not retrieve account from address: %v", err)
		return
	}

	//  this call blocks, up until Complete Transaction is called
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  from.Address,
		To:    &to.Address,
		Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("Test failed: cannot send transaction: %v", err)
	}

	if !reflect.DeepEqual(txHash, txHashCheck) {
		t.Errorf("Transaction hash returned from SendTransaction is invalid: expected %s, got %s", txHashCheck, txHash)
		return
	}

	time.Sleep(10 * time.Second)

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

	accountManager, err := geth.GetNodeManager().AccountManager()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	// create an account
	address, _, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}

	// obtain reference to status backend
	lightEthereum, err := geth.GetNodeManager().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txId string
	txFailedEventCalled := false
	txHash := common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txId = event["id"].(string)
			t.Logf("transaction queued (will be failed and completed on the second call): {id: %s}\n", txId)

			// try with wrong password
			// make sure that tx is NOT removed from the queue (by re-trying with the correct password)
			if _, err = geth.CompleteTransaction(txId, testAddressPassword+"wrong"); err == nil {
				t.Error("expects wrong password error, but call succeeded")
				return
			}

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			if txCount := backend.TransactionQueue().Count(); txCount != 1 {
				t.Errorf("txqueue cannot be empty, as tx has failed: expected = 1, got = %d", txCount)
				return
			}

			// now try to complete transaction, but with the correct password
			t.Log("allow 5 seconds before sedning the second CompleteTransaction")
			time.Sleep(5 * time.Second)
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testAddressPassword); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
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

	// send from the same test account (which is guaranteed to have ether)
	from, err := utils.MakeAddress(accountManager, testAddress)
	if err != nil {
		t.Errorf("could not retrieve account from address: %v", err)
		return
	}

	to, err := utils.MakeAddress(accountManager, address)
	if err != nil {
		t.Errorf("could not retrieve account from address: %v", err)
		return
	}

	//  this call blocks, and should return on *second* attempt to CompleteTransaction (w/ the correct password)
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  from.Address,
		To:    &to.Address,
		Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("cannot send transaction: %v", err)
		return
	}

	if !reflect.DeepEqual(txHash, txHashCheck) {
		t.Errorf("tx hash returned from SendTransaction is invalid: expected %s, got %s", txHashCheck, txHash)
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

	t.Log("sleep extra time, to allow sync")
	time.Sleep(5 * time.Second)
}

func TestNonExistentQueuedTransactions(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txHash = common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("Transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			// next call is the very same one, but with the correct password
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testAddressPassword); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
				return
			}

			t.Logf("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
		}
	})

	// try completing non-existing transaction
	if _, err = geth.CompleteTransaction("some-bad-transaction-id", testAddressPassword); err == nil {
		t.Error("error expected and not recieved")
		return
	}
	if err != status.ErrQueuedTxIdNotFound {
		t.Errorf("unexpected error recieved: expected '%s', got: '%s'", status.ErrQueuedTxIdNotFound.Error(), err.Error())
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
	lightEthereum, err := geth.GetNodeManager().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestQueuedTransactions")

	// replace transaction notification handler
	var txHash = common.Hash{}
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.GethEvent
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("Transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			// next call is the very same one, but with the correct password
			if txHash, err = geth.CompleteTransaction(event["id"].(string), testAddressPassword); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
				return
			}

			t.Logf("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
		}
	})

	txQueue := backend.TransactionQueue()
	var i = 0
	txIds := [status.DefaultTxQueueCap + 5 + 10]status.QueuedTxId{}
	backend.SetTransactionQueueHandler(func(queuedTx status.QueuedTx) {
		t.Logf("%d. Transaction queued (queue size: %d): {id: %v}\n", i, txQueue.Count(), queuedTx.Id)
		txIds[i] = queuedTx.Id
		i++
	})

	if txQueue.Count() != 0 {
		t.Errorf("transaction count should be zero: %d", txQueue.Count())
		return
	}

	for i := 0; i < 10; i++ {
		go backend.SendTransaction(nil, status.SendTxArgs{})
	}
	time.Sleep(3 * time.Second)

	t.Logf("Number of transactions queued: %d. Queue size (shouldn't be more than %d): %d", i, status.DefaultTxQueueCap, txQueue.Count())

	if txQueue.Count() != 10 {
		t.Errorf("transaction count should be 10: got %d", txQueue.Count())
		return
	}

	for i := 0; i < status.DefaultTxQueueCap+5; i++ { // stress test by hitting with lots of goroutines
		go backend.SendTransaction(nil, status.SendTxArgs{})
	}
	time.Sleep(5 * time.Second)

	if txQueue.Count() > status.DefaultTxQueueCap {
		t.Errorf("transaction count should be %d (or %d): got %d", status.DefaultTxQueueCap, status.DefaultTxQueueCap-1, txQueue.Count())
		return
	}

	for _, txId := range txIds {
		txQueue.Remove(txId)
	}

	if txQueue.Count() != 0 {
		t.Errorf("transaction count should be zero: %d", txQueue.Count())
		return
	}
}
