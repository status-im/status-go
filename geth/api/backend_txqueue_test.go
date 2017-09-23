package api_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/status-im/status-go/geth/txqueue"
)

// FIXME(tiabc): Sometimes it fails due to "no suitable peers found".
func (s *BackendTestSuite) TestSendContractTx() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	sampleAddress, _, _, err := s.backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash gethcommon.Hash
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint :dupl
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			log.Info("trying to complete with no user logged in")
			txHash, err = s.backend.CompleteTransaction(
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
			err = s.backend.AccountManager().SelectAccount(sampleAddress, TestConfig.Account1.Password)
			s.NoError(err)
			txHash, err = s.backend.CompleteTransaction(
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
			s.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
			txHash, err = s.backend.CompleteTransaction(
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
	require.NoError(err)

	txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
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

func (s *BackendTestSuite) TestSendEtherTx() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	// create an account
	sampleAddress, _, _, err := s.backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash = gethcommon.Hash{}
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint: dupl
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			log.Info("trying to complete with no user logged in")
			txHash, err = s.backend.CompleteTransaction(
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
			err = s.backend.AccountManager().SelectAccount(sampleAddress, TestConfig.Account1.Password)
			s.NoError(err)
			txHash, err = s.backend.CompleteTransaction(
				common.QueuedTxID(event["id"].(string)), TestConfig.Account1.Password)
			s.EqualError(
				err,
				txqueue.ErrInvalidCompleteTxSender.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]),
			)

			// the third call will work as expected (as we are logged in with correct credentials)
			log.Info("trying to complete with correct user, this should succeed")
			s.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
			txHash, err = s.backend.CompleteTransaction(
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
	txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
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
	s.Zero(s.backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *BackendTestSuite) TestSendEtherTxUpstream() {
	s.StartTestBackend(params.RopstenNetworkID, WithUpstream("https://ropsten.infura.io/z6GCTmjdP3FETEJmMBI4"))
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	err := s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	completeQueuedTransaction := make(chan struct{})

	// replace transaction notification handler
	var txHash = gethcommon.Hash{}
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint: dupl
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, "cannot unmarshal JSON: %s", jsonEvent)

		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			txHash, err = s.backend.CompleteTransaction(
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
	txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
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
	s.Zero(s.backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *BackendTestSuite) TestDoubleCompleteQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

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
			_, err = s.backend.CompleteTransaction(txID, TestConfig.Account1.Password+"wrong")
			s.EqualError(err, keystore.ErrDecrypt.Error())

			s.Equal(1, s.TxQueueManager().TransactionQueue().Count(), "txqueue cannot be empty, as tx has failed")

			// now try to complete transaction, but with the correct password
			txHash, err = s.backend.CompleteTransaction(txID, TestConfig.Account1.Password)
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
	txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
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
	s.Zero(s.backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
	s.True(txFailedEventCalled, "expected tx failure signal is not received")
}

func (s *BackendTestSuite) TestDiscardQueuedTransaction() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	// reset queue
	s.backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

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

			s.True(s.backend.TxQueueManager().TransactionQueue().Has(txID), "txqueue should still have test tx")

			// discard
			err := s.backend.DiscardTransaction(txID)
			s.NoError(err, "cannot discard tx")

			// try completing discarded transaction
			_, err = s.backend.CompleteTransaction(txID, TestConfig.Account1.Password)
			s.EqualError(err, "transaction hash not found", "expects tx not found, but call to CompleteTransaction succeeded")

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			s.False(s.backend.TxQueueManager().TransactionQueue().Has(txID),
				fmt.Sprintf("txqueue should not have test tx at this point (it should be discarded): %s", txID))

			close(completeQueuedTransaction)
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := txqueue.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			txFailedEventCalled = true
		}
	})

	// this call blocks, and should return when DiscardQueuedTransaction() is called
	txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.EqualError(err, txqueue.ErrQueuedTxDiscarded.Error(), "transaction is expected to be discarded")

	select {
	case <-completeQueuedTransaction:
	case <-time.After(time.Minute):
		s.FailNow("test timed out")
	}

	s.True(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned hash, while it shouldn't")
	s.Zero(s.backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
	s.True(txFailedEventCalled, "expected tx failure signal is not received")
}

func (s *BackendTestSuite) TestCompleteMultipleQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	// allow to sync
	time.Sleep(TestConfig.Node.SyncSeconds * time.Second)

	s.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	err := s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(err)

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
		txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
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
		results := s.backend.CompleteTransactions(txIDs, TestConfig.Account1.Password)
		require.Len(results, testTxCount+1)
		require.EqualError(results["invalid-tx-id"].Error, "transaction hash not found")

		for txID, txResult := range results {
			require.False(
				txResult.Error != nil && txID != "invalid-tx-id",
				"invalid error for %s", txID,
			)
			require.False(
				txResult.Hash == (gethcommon.Hash{}) && txID != "invalid-tx-id",
				"invalid hash (expected non empty hash): %s", txID,
			)
			log.Info("transaction complete", "URL", "https://ropsten.etherscan.io/tx/"+txResult.Hash.Hex())
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates

		for _, txID := range txIDs {
			require.False(
				s.backend.TxQueueManager().TransactionQueue().Has(txID),
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
	case <-time.After(30 * time.Second):
		s.FailNow("test timed out")
	}

	require.Zero(s.TxQueueManager().TransactionQueue().Count(), "queue should be empty")
}

func (s *BackendTestSuite) TestDiscardMultipleQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	// reset queue
	s.backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	testTxCount := 3
	txIDs := make(chan common.QueuedTxID, testTxCount)
	allTestTxDiscarded := make(chan struct{})

	// replace transaction notification handler
	txFailedEventCallCount := 0
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID := common.QueuedTxID(event["id"].(string))
			log.Info("transaction queued (will be discarded soon)", "id", txID)

			s.True(s.backend.TxQueueManager().TransactionQueue().Has(txID),
				"txqueue should still have test tx")
			txIDs <- txID
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := txqueue.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			txFailedEventCallCount++
			if txFailedEventCallCount == testTxCount {
				close(allTestTxDiscarded)
			}
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := s.backend.SendTransaction(nil, common.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		s.EqualError(err, txqueue.ErrQueuedTxDiscarded.Error())

		s.True(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned hash, while it shouldn't")
	}

	// wait for transactions, and discard immediately
	discardTxs := func(txIDs []common.QueuedTxID) {
		txIDs = append(txIDs, "invalid-tx-id")

		// discard
		discardResults := s.backend.DiscardTransactions(txIDs)
		require.Len(discardResults, 1, "cannot discard txs: %v", discardResults)
		require.Error(discardResults["invalid-tx-id"].Error, "transaction hash not found", "cannot discard txs: %v", discardResults)

		// try completing discarded transaction
		completeResults := s.backend.CompleteTransactions(txIDs, TestConfig.Account1.Password)
		require.Len(completeResults, testTxCount+1, "unexpected number of errors (call to CompleteTransaction should not succeed)")

		for _, txResult := range completeResults {
			require.Error(txResult.Error, "transaction hash not found", "invalid error for %s", txResult.Hash.Hex())
			require.Equal("0x0000000000000000000000000000000000000000000000000000000000000000", txResult.Hash.Hex(), "invalid hash (expected zero): %s", txResult.Hash.Hex())
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates

		for _, txID := range txIDs {
			require.False(
				s.backend.TxQueueManager().TransactionQueue().Has(txID),
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
		require.FailNow("test timed out")
	}

	require.Zero(s.backend.TxQueueManager().TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *BackendTestSuite) TestNonExistentQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(string) {})

	// try completing non-existing transaction
	_, err := s.backend.CompleteTransaction("some-bad-transaction-id", TestConfig.Account1.Password)
	s.Error(err, "error expected and not received")
	s.EqualError(err, txqueue.ErrQueuedTxIDNotFound.Error())
}

func (s *BackendTestSuite) TestEvictionOfQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	// reset queue
	s.backend.TxQueueManager().TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	queue := s.backend.TxQueueManager().TransactionQueue()
	var i = 0
	txIDs := [txqueue.DefaultTxQueueCap + 5 + 10]common.QueuedTxID{}
	s.backend.TxQueueManager().SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		log.Info("tx enqueued", "i", i+1, "queue size", queue.Count(), "id", queuedTx.ID)
		txIDs[i] = queuedTx.ID
		i++
	})

	s.Zero(queue.Count(), "transaction count should be zero")

	for i := 0; i < 10; i++ {
		go s.backend.SendTransaction(nil, common.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(2 * time.Second) // FIXME(tiabc): more reliable synchronization to ensure all transactions are enqueued

	log.Info(fmt.Sprintf("Number of transactions queued: %d. Queue size (shouldn't be more than %d): %d",
		i, txqueue.DefaultTxQueueCap, queue.Count()))

	s.Equal(10, queue.Count(), "transaction count should be 10")

	for i := 0; i < txqueue.DefaultTxQueueCap+5; i++ { // stress test by hitting with lots of goroutines
		go s.backend.SendTransaction(nil, common.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(3 * time.Second)

	require.True(queue.Count() <= txqueue.DefaultTxQueueCap, "transaction count should be %d (or %d): got %d", txqueue.DefaultTxQueueCap, txqueue.DefaultTxQueueCap-1, queue.Count())

	for _, txID := range txIDs {
		queue.Remove(txID)
	}

	require.Zero(queue.Count(), "transaction count should be zero: %d", queue.Count())
}
