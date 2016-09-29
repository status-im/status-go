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
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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

	// test transaction queueing
	lightEthereum, err := geth.GetNodeManager().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return
	}
	backend := lightEthereum.StatusBackend

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan bool, 1)
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
			glog.V(logger.Info).Infof("Transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			if txHash, err = geth.CompleteTransaction(event["id"].(string), testAddressPassword); err != nil {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], err)
				return
			}

			glog.V(logger.Info).Infof("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			completeQueuedTransaction <- true // so that timeout is aborted
		}
	})

	// try completing non-existing transaction
	if _, err := geth.CompleteTransaction("some-bad-transaction-id", testAddressPassword); err == nil {
		t.Error("error expected and not recieved")
		return
	}

	// send normal transaction
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

	// now test eviction queue
	txQueue := backend.TransactionQueue()
	var i = 0
	backend.SetTransactionQueueHandler(func(queuedTx status.QueuedTx) {
		//glog.V(logger.Info).Infof("%d. Transaction queued (queue size: %d): {id: %v}\n", i, txQueue.Count(), queuedTx.Id)
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

	if txQueue.Count() != status.DefaultTxQueueCap && txQueue.Count() != (status.DefaultTxQueueCap-1) {
		t.Errorf("transaction count should be %d (or %d): got %d", status.DefaultTxQueueCap, status.DefaultTxQueueCap-1, txQueue.Count())
		return
	}
}
