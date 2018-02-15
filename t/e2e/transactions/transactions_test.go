package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/transactions"
	"github.com/status-im/status-go/geth/transactions/queue"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
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

		if sg.Type == transactions.EventTransactionQueued {
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

		if signalEnvelope.Type == transactions.EventTransactionQueued {
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

		if envelope.Type == transactions.EventTransactionQueued {
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
				queue.ErrInvalidCompleteTxSender.Error(),
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

	gas := uint64(params.DefaultGas)
	txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   nil, // marker, contract creation is expected
		//Value: (*hexutil.Big)(new(big.Int).Mul(big.NewInt(1), gethcommon.Ether)),
		Gas:   (*hexutil.Uint64)(&gas),
		Input: (*hexutil.Bytes)(&byteCode),
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

		if envelope.Type == transactions.EventTransactionQueued {
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
				queue.ErrInvalidCompleteTxSender.Error(),
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

		if envelope.Type == transactions.EventTransactionQueued {
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

		if envelope.Type == transactions.EventTransactionQueued {
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

		if envelope.Type == transactions.EventTransactionFailed {
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

		if envelope.Type == transactions.EventTransactionQueued {
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

		if envelope.Type == transactions.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := transactions.ErrQueuedTxDiscarded.Error()
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
	s.EqualError(err, transactions.ErrQueuedTxDiscarded.Error(), "transaction is expected to be discarded")

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
	s.setupLocalNode()
	defer s.StopTestBackend()

	s.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	err := s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	s.sendConcurrentTransactions(3)
}

func (s *TransactionsTestSuite) TestDiscardMultipleQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())

	// reset queue
	s.Backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	testTxCount := 3
	txIDs := make(chan common.QueuedTxID, testTxCount)
	allTestTxDiscarded := make(chan struct{})

	// replace transaction notification handler
	txFailedEventCallCount := 0
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)
		if envelope.Type == transactions.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID := common.QueuedTxID(event["id"].(string))
			log.Info("transaction queued (will be discarded soon)", "id", txID)

			s.True(s.Backend.TxQueueManager().TransactionQueue().Has(txID),
				"txqueue should still have test tx")
			txIDs <- txID
		}

		if envelope.Type == transactions.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := transactions.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			txFailedEventCallCount++
			if txFailedEventCallCount == testTxCount {
				close(allTestTxDiscarded)
			}
		}
	})

	require := s.Require()

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		require.EqualError(err, transactions.ErrQueuedTxDiscarded.Error())
		require.Equal(gethcommon.Hash{}, txHashCheck, "transaction returned hash, while it shouldn't")
	}

	txQueueManager := s.Backend.TxQueueManager()

	// wait for transactions, and discard immediately
	discardTxs := func(txIDs []common.QueuedTxID) {
		txIDs = append(txIDs, "invalid-tx-id")

		// discard
		discardResults := txQueueManager.DiscardTransactions(txIDs)
		require.Len(discardResults, 1, "cannot discard txs: %v", discardResults)
		require.Error(discardResults["invalid-tx-id"].Error, "transaction hash not found", "cannot discard txs: %v", discardResults)

		// try completing discarded transaction
		completeResults := txQueueManager.CompleteTransactions(txIDs, TestConfig.Account1.Password)
		require.Len(completeResults, testTxCount+1, "unexpected number of errors (call to CompleteTransaction should not succeed)")

		for _, txResult := range completeResults {
			require.Error(txResult.Error, "transaction hash not found", "invalid error for %s", txResult.Hash.Hex())
			require.Equal(gethcommon.Hash{}, txResult.Hash, "invalid hash (expected zero): %s", txResult.Hash.Hex())
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates

		for _, txID := range txIDs {
			require.False(
				txQueueManager.TransactionQueue().Has(txID),
				"txqueue should not have test tx at this point (it should be discarded): %s",
				txID,
			)
		}
	}
	go func() {
		ids := make([]common.QueuedTxID, testTxCount)
		for i := 0; i < testTxCount; i++ {
			ids[i] = <-txIDs
		}

		discardTxs(ids)
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		go sendTx()
	}

	select {
	case <-allTestTxDiscarded:
	case <-time.After(1 * time.Minute):
		s.FailNow("test timed out")
	}

	s.Zero(s.Backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *TransactionsTestSuite) TestNonExistentQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(string) {})

	// try completing non-existing transaction
	_, err := s.Backend.CompleteTransaction("some-bad-transaction-id", TestConfig.Account1.Password)
	s.Error(err, "error expected and not received")
	s.EqualError(err, queue.ErrQueuedTxIDNotFound.Error())
}

func (s *TransactionsTestSuite) TestEvictionOfQueuedTransactions() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	var m sync.Mutex
	txCount := 0
	txIDs := [queue.DefaultTxQueueCap + 5 + 10]common.QueuedTxID{}

	signal.SetDefaultNodeNotificationHandler(func(rawSignal string) {
		var sg signal.Envelope
		err := json.Unmarshal([]byte(rawSignal), &sg)
		s.NoError(err)

		if sg.Type == transactions.EventTransactionQueued {
			event := sg.Event.(map[string]interface{})
			txID := event["id"].(string)
			m.Lock()
			txIDs[txCount] = common.QueuedTxID(txID)
			txCount++
			m.Unlock()
		}
	})

	// reset queue
	s.Backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	s.NoError(s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	txQueue := s.Backend.TxQueueManager().TransactionQueue()
	s.Zero(txQueue.Count(), "transaction count should be zero")

	for j := 0; j < 10; j++ {
		go s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(2 * time.Second)
	s.Equal(10, txQueue.Count(), "transaction count should be 10")

	for i := 0; i < queue.DefaultTxQueueCap+5; i++ { // stress test by hitting with lots of goroutines
		go s.Backend.SendTransaction(context.TODO(), common.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(5 * time.Second)

	s.True(txQueue.Count() <= queue.DefaultTxQueueCap, "transaction count should be %d (or %d): got %d", queue.DefaultTxQueueCap, queue.DefaultTxQueueCap-1, txQueue.Count())

	m.Lock()
	for _, txID := range txIDs {
		txQueue.Remove(txID)
	}
	m.Unlock()
	s.Zero(txQueue.Count(), "transaction count should be zero: %d", txQueue.Count())
}

func (s *TransactionsTestSuite) TestCompleteMultipleQueuedTransactionsUpstream() {
	s.setupUpstreamNode()
	defer s.StopTestBackend()

	s.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	err := s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	s.sendConcurrentTransactions(30)
}

func (s *TransactionsTestSuite) setupLocalNode() {
	s.StartTestBackend()

	EnsureNodeSync(s.Backend.NodeManager())
}

func (s *TransactionsTestSuite) setupUpstreamNode() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	addr, err := GetRemoteURL()
	s.NoError(err)
	s.StartTestBackend(e2e.WithUpstream(addr))
}

func (s *TransactionsTestSuite) sendConcurrentTransactions(testTxCount int) {
	txIDs := make(chan common.QueuedTxID, testTxCount)
	allTestTxCompleted := make(chan struct{})

	require := s.Require()

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		require.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == transactions.EventTransactionQueued {
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
		require.NoError(err, "cannot send transaction")
		require.NotEqual(gethcommon.Hash{}, txHashCheck, "transaction returned empty hash")
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
	go func() {
		ids := make([]common.QueuedTxID, testTxCount)
		for i := 0; i < testTxCount; i++ {
			ids[i] = <-txIDs
		}

		completeTxs(ids)
		close(allTestTxCompleted)
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		go sendTx()
	}

	select {
	case <-allTestTxCompleted:
	case <-time.After(50 * time.Second):
		s.FailNow("test timed out")
	}

	s.Zero(s.TxQueueManager().TransactionQueue().Count(), "queue should be empty")
}
