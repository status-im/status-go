package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/txqueue"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

func TestTransactionsTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionsTestSuite))
}

type TransactionsTestSuite struct {
	e2e.BackendTestSuite
}

func (s *TransactionsTestSuite) TestCallRPCSendTransaction() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	err := s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	transactionCompleted := make(chan struct{})

	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var sg signal.Envelope
		err := json.Unmarshal([]byte(rawSignal), &sg)
		s.NoError(err)

		if sg.Type == txqueue.EventTransactionQueued {
			event := sg.Event.(map[string]interface{})
			txID := event["id"].(string)
			txHash, err = s.Backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account1.Password)
			s.NoError(err, "cannot complete queued transaction %s", txID)

			close(transactionCompleted)
		}
	})

	result := s.Backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account1.Address + `",
			"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
			"value": "0x9184e72a"
		}]
	}`)
	s.NotContains(result, "error")

	select {
	case <-transactionCompleted:
	case <-time.After(time.Minute):
		s.FailNow("sending transaction timed out")
	}

	s.Equal(`{"jsonrpc":"2.0","id":1,"result":"`+txHash.String()+`"}`, result)
}

func (s *TransactionsTestSuite) TestCallRPCSendTransactionUpstream() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	addr, err := GetRemoteURL()
	s.NoError(err)
	s.StartTestBackend(e2e.WithUpstream(addr))
	defer s.StopTestBackend()

	err = s.Backend.AccountManager().SelectAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	s.NoError(err)

	transactionCompleted := make(chan struct{})

	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var signalEnvelope signal.Envelope
		err := json.Unmarshal([]byte(rawSignal), &signalEnvelope)
		s.NoError(err)

		if signalEnvelope.Type == txqueue.EventTransactionQueued {
			event := signalEnvelope.Event.(map[string]interface{})
			txID := event["id"].(string)

			// Complete with a wrong passphrase.
			txHash, err = s.Backend.CompleteTransaction(common.QueuedTxID(txID), "some-invalid-passphrase")
			s.EqualError(err, keystore.ErrDecrypt.Error(), "should return an error as the passphrase was invalid")

			// Complete with a correct passphrase.
			txHash, err = s.Backend.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account2.Password)
			s.NoError(err, "cannot complete queued transaction %s", txID)

			close(transactionCompleted)
		}
	})

	result := s.Backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account2.Address + `",
			"to": "` + TestConfig.Account1.Address + `",
			"value": "0x9184e72a"
		}]
	}`)
	s.NotContains(result, "error")

	select {
	case <-transactionCompleted:
	case <-time.After(time.Minute):
		s.FailNow("sending transaction timed out")
	}

	s.Equal(`{"jsonrpc":"2.0","id":1,"result":"`+txHash.String()+`"}`, result)
}

// FIXME(tiabc): Sometimes it fails due to "no suitable peers found".
func (s *TransactionsTestSuite) TestSendContractTx() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	sampleAddress, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint :dupl
		var envelope signal.Envelope
		err = json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			log.Info("trying to complete with no user logged in")
			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)),
				TestConfig.Account1.Password,
			)
			s.EqualError(
				err,
				account.ErrNoAccountSelected.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]),
			)

			// the second call will also fail (we are logged in as different user)
			log.Info("trying to complete with invalid user")
			err = s.Backend.AccountManager().SelectAccount(sampleAddress, TestConfig.Account1.Password)
			s.NoError(err)
			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)),
				TestConfig.Account1.Password,
			)
			s.EqualError(
				err,
				txqueue.ErrInvalidCompleteTxSender.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]),
			)

			// the third call will work as expected (as we are logged in with correct credentials)
			log.Info("trying to complete with correct user, this should succeed")
			s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)),
				TestConfig.Account1.Password,
			)
			s.NoError(err, fmt.Sprintf("cannot complete queued transaction[%v]", event["id"]))

			log.Info("contract transaction complete", "URL", "https://ropsten.etherscan.io/tx/"+txHash.Hex())
			close(completeQueuedTransaction)
			return
		}
	})

	// this call blocks, up until Complete Transaction is called
	byteCode, err := hexutil.Decode(`0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029`)
	s.NoError(err)

	txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   nil, // marker, contract creation is expected
		//Value: (*hexutil.Big)(new(big.Int).Mul(big.NewInt(1), gethcommon.Ether)),
		Gas:  (*hexutil.Big)(big.NewInt(params.DefaultGas)),
		Data: byteCode,
	})
	s.NoError(err, "cannot send transaction")

	select {
	case <-completeQueuedTransaction:
	case <-time.After(2 * time.Minute):
		s.FailNow("completing transaction timed out")
	}

	s.Equal(txHashCheck.Hex(), txHash.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction was never queued or completed")
	s.Zero(s.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *TransactionsTestSuite) TestSendEther() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	backend := s.LightEthereumService().StatusBackend
	s.NotNil(backend)

	// create an account
	sampleAddress, _, _, err := s.Backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	s.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash = gethcommon.Hash{}
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint: dupl
		var envelope signal.Envelope
		err = json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			log.Info("trying to complete with no user logged in")
			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)),
				TestConfig.Account1.Password,
			)
			s.EqualError(
				err,
				account.ErrNoAccountSelected.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]),
			)

			// the second call will also fail (we are logged in as different user)
			log.Info("trying to complete with invalid user")
			err = s.Backend.AccountManager().SelectAccount(sampleAddress, TestConfig.Account1.Password)
			s.NoError(err)
			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)), TestConfig.Account1.Password)
			s.EqualError(
				err,
				txqueue.ErrInvalidCompleteTxSender.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]),
			)

			// the third call will work as expected (as we are logged in with correct credentials)
			log.Info("trying to complete with correct user, this should succeed")
			s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)),
				TestConfig.Account1.Password,
			)
			s.NoError(err, fmt.Sprintf("cannot complete queued transaction[%v]", event["id"]))

			close(completeQueuedTransaction)
			return
		}
	})

	// this call blocks, up until Complete Transaction is called
	txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.NoError(err, "cannot send transaction")

	select {
	case <-completeQueuedTransaction:
	case <-time.After(2 * time.Minute):
		s.FailNow("completing transaction timed out")
	}

	s.Equal(txHashCheck.Hex(), txHash.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction was never queued or completed")
	s.Zero(s.Backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *TransactionsTestSuite) TestSendEtherTxUpstream() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	addr, err := GetRemoteURL()
	s.NoError(err)
	s.StartTestBackend(e2e.WithUpstream(addr))
	defer s.StopTestBackend()

	err = s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash = gethcommon.Hash{}
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint: dupl
		var envelope signal.Envelope
		err = json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, "cannot unmarshal JSON: %s", jsonEvent)

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			txHash, err = s.Backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)),
				TestConfig.Account1.Password,
			)
			s.NoError(err, "cannot complete queued transaction[%v]", event["id"])

			log.Info("contract transaction complete", "URL", "https://ropsten.etherscan.io/tx/"+txHash.Hex())
			close(completeQueuedTransaction)
		}
	})

	// This call blocks, up until Complete Transaction is called.
	// Explicitly not setting Gas to get it estimated.
	txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
		From:     common.FromAddress(TestConfig.Account1.Address),
		To:       common.ToAddress(TestConfig.Account2.Address),
		GasPrice: (*hexutil.Big)(big.NewInt(28000000000)),
		Value:    (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.NoError(err, "cannot send transaction")

	select {
	case <-completeQueuedTransaction:
	case <-time.After(1 * time.Minute):
		s.FailNow("completing transaction timed out")
	}

	s.Equal(txHash.Hex(), txHashCheck.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.Zero(s.Backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *TransactionsTestSuite) TestDoubleCompleteQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	backend := s.LightEthereumService().StatusBackend
	s.NotNil(backend)

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	txFailedEventCalled := false
	txHash := gethcommon.Hash{}
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID := common.QueuedTxID(event["id"].(string))
			log.Info("transaction queued (will be failed and completed on the second call)", "id", txID)

			// try with wrong password
			// make sure that tx is NOT removed from the queue (by re-trying with the correct password)
			_, err = s.Backend.CompleteTransaction(txID, TestConfig.Account1.Password+"wrong")
			s.EqualError(err, keystore.ErrDecrypt.Error())

			s.Equal(1, s.TxQueueManager().TransactionQueue().Count(), "txqueue cannot be empty, as tx has failed")

			// now try to complete transaction, but with the correct password
			txHash, err = s.Backend.CompleteTransaction(txID, TestConfig.Account1.Password)
			s.NoError(err)

			log.Info("transaction complete", "URL", "https://rinkeby.etherscan.io/tx/"+txHash.Hex())
			close(completeQueuedTransaction)
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := "could not decrypt key with given passphrase"
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("2", receivedErrCode)

			txFailedEventCalled = true
		}
	})

	// this call blocks, and should return on *second* attempt to CompleteTransaction (w/ the correct password)
	txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.NoError(err, "cannot send transaction")

	select {
	case <-completeQueuedTransaction:
	case <-time.After(time.Minute):
		s.FailNow("test timed out")
	}

	s.Equal(txHashCheck.Hex(), txHash.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction was never queued or completed")
	s.Zero(s.Backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
	s.True(txFailedEventCalled, "expected tx failure signal is not received")
}

func (s *TransactionsTestSuite) TestDiscardQueuedTransaction() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	backend := s.LightEthereumService().StatusBackend
	s.NotNil(backend)

	// reset queue
	s.Backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	txFailedEventCalled := false
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID := common.QueuedTxID(event["id"].(string))
			log.Info("transaction queued (will be discarded soon)", "id", txID)

			s.True(s.Backend.TxQueueManager().TransactionQueue().Has(txID), "txqueue should still have test tx")

			// discard
			err := s.Backend.DiscardTransaction(txID)
			s.NoError(err, "cannot discard tx")

			// try completing discarded transaction
			_, err = s.Backend.CompleteTransaction(txID, TestConfig.Account1.Password)
			s.EqualError(err, "transaction hash not found", "expects tx not found, but call to CompleteTransaction succeeded")

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			s.False(s.Backend.TxQueueManager().TransactionQueue().Has(txID),
				fmt.Sprintf("txqueue should not have test tx at this point (it should be discarded): %s", txID))

			close(completeQueuedTransaction)
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := common.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			txFailedEventCalled = true
		}
	})

	// this call blocks, and should return when DiscardQueuedTransaction() is called
	txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.EqualError(err, common.ErrQueuedTxDiscarded.Error(), "transaction is expected to be discarded")

	select {
	case <-completeQueuedTransaction:
	case <-time.After(time.Minute):
		s.FailNow("test timed out")
	}

	s.True(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned hash, while it shouldn't")
	s.Zero(s.Backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
	s.True(txFailedEventCalled, "expected tx failure signal is not received")
}

func (s *TransactionsTestSuite) TestCompleteMultipleQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())
	s.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	err := s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	testTxCount := 3
	txIDs := make(chan common.QueuedTxID, testTxCount)
	allTestTxCompleted := make(chan struct{})

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID := common.QueuedTxID(event["id"].(string))
			log.Info("transaction queued (will be completed in a single call, once aggregated)", "id", txID)

			txIDs <- txID
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		s.NoError(err, "cannot send transaction")
		s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned empty hash")
	}

	// wait for transactions, and complete them in a single call
	completeTxs := func(txIDs []common.QueuedTxID) {
		txIDs = append(txIDs, "invalid-tx-id")
		results := s.Backend.CompleteTransactions(txIDs, TestConfig.Account1.Password)
		s.Len(results, testTxCount+1)
		s.EqualError(results["invalid-tx-id"].Error, "transaction hash not found")

		for txID, txResult := range results {
			s.False(
				txResult.Error != nil && txID != "invalid-tx-id",
				"invalid error for %s", txID,
			)
			s.False(
				txResult.Hash == (gethcommon.Hash{}) && txID != "invalid-tx-id",
				"invalid hash (expected non empty hash): %s", txID,
			)
			log.Info("transaction complete", "URL", "https://ropsten.etherscan.io/tx/"+txResult.Hash.Hex())
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates

		for _, txID := range txIDs {
			s.False(
				s.Backend.TxQueueManager().TransactionQueue().Has(txID),
				"txqueue should not have test tx at this point (it should be completed)",
			)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		ids := make([]common.QueuedTxID, testTxCount)
		for i := 0; i < testTxCount; i++ {
			ids[i] = <-txIDs
		}

		completeTxs(ids)
		close(allTestTxCompleted)
		wg.Done()
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		wg.Add(1)
		go func() {
			sendTx()
			wg.Done()
		}()
	}

	select {
	case <-allTestTxCompleted:
	case <-time.After(30 * time.Second):
		s.FailNow("test timed out")
	}
	wg.Wait()

	s.Zero(s.TxQueueManager().TransactionQueue().Count(), "queue should be empty")
}

func (s *TransactionsTestSuite) TestDiscardMultipleQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	backend := s.LightEthereumService().StatusBackend
	s.NotNil(backend)

	// reset queue
	s.Backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	testTxCount := 3
	txIDs := make(chan common.QueuedTxID, testTxCount)
	allTestTxDiscarded := make(chan struct{})

	// replace transaction notification handler
	txFailedEventCallCount := int32(0)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID := common.QueuedTxID(event["id"].(string))
			log.Info("transaction queued (will be discarded soon)", "id", txID)

			s.True(s.Backend.TxQueueManager().TransactionQueue().Has(txID),
				"txqueue should still have test tx")
			txIDs <- txID
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := common.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			if int(atomic.AddInt32(&txFailedEventCallCount, 1)) == testTxCount {
				close(allTestTxDiscarded)
			}
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		s.EqualError(err, common.ErrQueuedTxDiscarded.Error())

		s.True(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned hash, while it shouldn't")
	}

	// wait for transactions, and discard immediately
	discardTxs := func(txIDList []common.QueuedTxID) {
		txIDList = append(txIDList, "invalid-tx-id")

		// discard
		discardResults := s.Backend.DiscardTransactions(txIDList)
		s.Len(discardResults, 1, "cannot discard txs: %v", discardResults)
		s.Error(discardResults["invalid-tx-id"].Error, "transaction hash not found", "cannot discard txs: %v", discardResults)

		// try completing discarded transaction
		completeResults := s.Backend.CompleteTransactions(txIDList, TestConfig.Account1.Password)
		s.Len(completeResults, testTxCount+1, "unexpected number of errors (call to CompleteTransaction should not succeed)")

		for _, txResult := range completeResults {
			s.Error(txResult.Error, "transaction hash not found", "invalid error for %s", txResult.Hash.Hex())
			s.Equal("0x0000000000000000000000000000000000000000000000000000000000000000", txResult.Hash.Hex(), "invalid hash (expected zero): %s", txResult.Hash.Hex())
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates

		for _, txID := range txIDList {
			s.False(
				s.Backend.TxQueueManager().TransactionQueue().Has(txID),
				"txqueue should not have test tx at this point (it should be discarded): %s",
				txID,
			)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		ids := make([]common.QueuedTxID, testTxCount)
		for i := 0; i < testTxCount; i++ {
			ids[i] = <-txIDs
		}

		discardTxs(ids)
		wg.Done()
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		wg.Add(1)
		go func() {
			sendTx()
			wg.Done()
		}()
	}

	select {
	case <-allTestTxDiscarded:
	case <-time.After(1 * time.Minute):
		s.FailNow("test timed out")
	}

	wg.Wait()
	s.Zero(s.Backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *TransactionsTestSuite) TestNonExistentQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	backend := s.LightEthereumService().StatusBackend
	s.NotNil(backend)

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(string) {})

	// try completing non-existing transaction
	_, err := s.Backend.CompleteTransaction("some-bad-transaction-id", TestConfig.Account1.Password)
	s.Error(err, "error expected and not received")
	s.EqualError(err, txqueue.ErrQueuedTxIDNotFound.Error())
}

func (s *TransactionsTestSuite) TestEvictionOfQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	backend := s.LightEthereumService().StatusBackend
	s.NotNil(backend)

	// reset queue
	s.Backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	txQueue := s.Backend.TxQueueManager().TransactionQueue()

	var i = int32(0)
	txIDs := [txqueue.DefaultTxQueueCap + 5 + 10]common.QueuedTxID{}
	s.Backend.TxQueueManager().SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		n := atomic.LoadInt32(&i)
		log.Info("tx enqueued", "i", n+1, "queue size", txQueue.Count(), "id", queuedTx.ID())
		txIDs[n] = queuedTx.ID()

		atomic.AddInt32(&i, 1)
	})

	s.Zero(txQueue.Count(), "transaction count should be zero")

	var wg sync.WaitGroup
	firstBatchSize := 10
	for j := 0; j < firstBatchSize; j++ {
		wg.Add(1)
		go func() {
			wg.Done()
			s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{}) // nolint: errcheck
		}()
	}
	ensureQueueTx(txQueue, firstBatchSize)

	log.Info(fmt.Sprintf("Number of transactions queued: %d. Queue size (shouldn't be more than %d): %d",
		atomic.LoadInt32(&i), txqueue.DefaultTxQueueCap, txQueue.Count()))

	s.Equal(10, txQueue.Count(), "transaction count should be 10")

	secondBatchSize := txqueue.DefaultTxQueueCap + 5
	for j := 0; j < secondBatchSize; j++ { // stress test by hitting with lots of goroutines
		wg.Add(1)
		go func() {
			wg.Done()
			s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{}) // nolint: errcheck
		}()
	}
	wg.Wait()
	ensureQueueTx(txQueue, txqueue.DefaultTxQueueCap-1)

	s.True(txQueue.Count() <= txqueue.DefaultTxQueueCap, "transaction count should be %d (or %d): got %d", txqueue.DefaultTxQueueCap, txqueue.DefaultTxQueueCap-1, txQueue.Count())

	for _, txID := range txIDs {
		txQueue.Remove(txID)
	}

	s.Zero(txQueue.Count(), "transaction count should be zero: %d", txQueue.Count())
}

func ensureQueueTx(txQueue common.TxQueue, n int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	safetyMargin := 3
	for range ticker.C {
		if txQueue.Count() == n {
			for i := 0; i < safetyMargin; i++ {
				<-ticker.C
			}
			return
		}
	}
}
