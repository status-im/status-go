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
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
)

func (s *BackendTestSuite) TestSendContractTx() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// create an account
	sampleAddress, _, _, err := s.backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 10)
	common.PanicAfter(20*time.Second, completeQueuedTransaction, s.T().Name())

	// replace transaction notification handler
	var txHash = gethcommon.Hash{}
	var txHashCheck = gethcommon.Hash{}
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint :dupl
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			log.Info("trying to complete with no user logged in")
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.EqualError(err, node.ErrNoAccountSelected.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]))

			// the second call will also fail (we are logged in as different user)
			log.Info("trying to complete with invalid user")
			err = s.backend.AccountManager().SelectAccount(sampleAddress, TestConfig.Account1.Password)
			s.NoError(err)
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.EqualError(err, status.ErrInvalidCompleteTxSender.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]))

			// the third call will work as expected (as we are logged in with correct credentials)
			log.Info("trying to complete with correct user, this should suceed")
			s.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.NoError(err, fmt.Sprintf("cannot complete queued transaction[%v]", event["id"]))

			log.Info("contract transaction complete", "URL", "https://rinkeby.etherscan.io/tx/"+txHash.Hex())
			close(completeQueuedTransaction)
			return
		}
	})

	// this call blocks, up until Complete Transaction is called
	byteCode, err := hexutil.Decode(`0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029`)
	require.NoError(err)

	// send transaction
	txHashCheck, err = backend.SendTransaction(nil, status.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   nil, // marker, contract creation is expected
		//Value: (*hexutil.Big)(new(big.Int).Mul(big.NewInt(1), gethcommon.Ether)),
		Gas:  (*hexutil.Big)(big.NewInt(params.DefaultGas)),
		Data: byteCode,
	})
	s.NoError(err, "cannot send transaction")

	<-completeQueuedTransaction
	s.Equal(txHashCheck.Hex(), txHash.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction was never queued or completed")
	s.Zero(backend.TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *BackendTestSuite) TestSendEtherTx() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// create an account
	sampleAddress, _, _, err := s.backend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	common.PanicAfter(20*time.Second, completeQueuedTransaction, s.T().Name())

	// replace transaction notification handler
	var txHash = gethcommon.Hash{}
	var txHashCheck = gethcommon.Hash{}
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) { // nolint: dupl
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction queued (will be completed shortly)", "id", event["id"].(string))

			// the first call will fail (we are not logged in, but trying to complete tx)
			log.Info("trying to complete with no user logged in")
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.EqualError(err, node.ErrNoAccountSelected.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]))

			// the second call will also fail (we are logged in as different user)
			log.Info("trying to complete with invalid user")
			err = s.backend.AccountManager().SelectAccount(sampleAddress, TestConfig.Account1.Password)
			s.NoError(err)
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.EqualError(err, status.ErrInvalidCompleteTxSender.Error(),
				fmt.Sprintf("expected error on queued transaction[%v] not thrown", event["id"]))

			// the third call will work as expected (as we are logged in with correct credentials)
			log.Info("trying to complete with correct user, this should suceed")
			s.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.NoError(err, fmt.Sprintf("cannot complete queued transaction[%v]", event["id"]))

			log.Info("contract transaction complete", "URL", "https://rinkeby.etherscan.io/tx/"+txHash.Hex())
			close(completeQueuedTransaction)
			return
		}
	})

	//  this call blocks, up until Complete Transaction is called
	txHashCheck, err = backend.SendTransaction(nil, status.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.NoError(err, "cannot send transaction")

	<-completeQueuedTransaction
	s.Equal(txHashCheck.Hex(), txHash.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction was never queued or completed")
	s.Zero(backend.TransactionQueue().Count(), "tx queue must be empty at this point")
}

func (s *BackendTestSuite) TestDoubleCompleteQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	common.PanicAfter(20*time.Second, completeQueuedTransaction, s.T().Name())

	// replace transaction notification handler
	var txID string
	txFailedEventCalled := false
	txHash := gethcommon.Hash{}
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			log.Info("transaction queued (will be failed and completed on the second call)", "id", txID)

			// try with wrong password
			// make sure that tx is NOT removed from the queue (by re-trying with the correct password)
			_, err = s.backend.CompleteTransaction(txID, TestConfig.Account1.Password+"wrong")
			s.EqualError(err, keystore.ErrDecrypt.Error())

			s.Equal(1, backend.TransactionQueue().Count(), "txqueue cannot be empty, as tx has failed")

			// now try to complete transaction, but with the correct password
			txHash, err = s.backend.CompleteTransaction(event["id"].(string), TestConfig.Account1.Password)
			s.NoError(err)

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			s.Equal(0, backend.TransactionQueue().Count(), "txqueue must be empty, as tx has completed")

			log.Info("transaction complete", "URL", "https://rinkeby.etherscan.io/tx/"+txHash.Hex())
			close(completeQueuedTransaction)
		}

		if envelope.Type == node.EventTransactionFailed {
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

	//  this call blocks, and should return on *second* attempt to CompleteTransaction (w/ the correct password)
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.NoError(err, "cannot send transaction")

	<-completeQueuedTransaction
	s.Equal(txHashCheck.Hex(), txHash.Hex(), "transaction hash returned from SendTransaction is invalid")
	s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction was never queued or completed")
	s.Zero(backend.TransactionQueue().Count(), "tx queue must be empty at this point")
	s.True(txFailedEventCalled, "expected tx failure signal is not received")
}

func (s *BackendTestSuite) TestDiscardQueuedTransaction() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	common.PanicAfter(30*time.Second, completeQueuedTransaction, s.T().Name())

	// replace transaction notification handler
	var txID string
	txFailedEventCalled := false
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))

		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			log.Info("transaction queued (will be discarded soon)", "id", txID)

			s.True(backend.TransactionQueue().Has(status.QueuedTxID(txID)), "txqueue should still have test tx")

			// discard
			err := s.backend.DiscardTransaction(txID)
			s.NoError(err, "cannot discard tx")

			// try completing discarded transaction
			_, err = s.backend.CompleteTransaction(txID, TestConfig.Account1.Password)
			s.EqualError(err, "transaction hash not found", "expects tx not found, but call to CompleteTransaction succeeded")

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			s.False(backend.TransactionQueue().Has(status.QueuedTxID(txID)),
				fmt.Sprintf("txqueue should not have test tx at this point (it should be discarded): %s", txID))

			close(completeQueuedTransaction)
		}

		if envelope.Type == node.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := status.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			txFailedEventCalled = true
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() is called
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	s.EqualError(err, status.ErrQueuedTxDiscarded.Error(), "transaction is expected to be discarded")

	<-completeQueuedTransaction
	s.True(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned hash, while it shouldn't")
	s.Zero(backend.TransactionQueue().Count(), "tx queue must be empty at this point")
	s.True(txFailedEventCalled, "expected tx failure signal is not received")
}

func (s *BackendTestSuite) TestCompleteMultipleQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIDs := make(chan string, testTxCount)
	allTestTxCompleted := make(chan struct{}, 1)

	// replace transaction notification handler
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txID string
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err, fmt.Sprintf("cannot unmarshal JSON: %s", jsonEvent))
		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			log.Info("transaction queued (will be completed in a single call, once aggregated)", "id", txID)

			txIDs <- txID
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		s.NoError(err, "cannot send transaction")
		s.False(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned empty hash")
	}

	// wait for transactions, and complete them in a single call
	completeTxs := func(txIDStrings string) {
		var parsedIDs []string
		err := json.Unmarshal([]byte(txIDStrings), &parsedIDs)
		s.NoError(err)

		parsedIDs = append(parsedIDs, "invalid-tx-id")
		updatedTxIDStrings, _ := json.Marshal(parsedIDs)

		// complete
		results := s.backend.CompleteTransactions(string(updatedTxIDStrings), TestConfig.Account1.Password)
		if len(results) != (testTxCount+1) || results["invalid-tx-id"].Error.Error() != "transaction hash not found" {
			s.Fail(fmt.Sprintf("cannot complete txs: %v", results))
			return
		}
		for txID, txResult := range results {
			if txResult.Error != nil && txID != "invalid-tx-id" {
				s.Fail(fmt.Sprintf("invalid error for %s", txID))
				return
			}
			if txResult.Hash.Hex() == "0x0000000000000000000000000000000000000000000000000000000000000000" && txID != "invalid-tx-id" {
				s.Fail(fmt.Sprintf("invalid hash (expected non empty hash): %s", txID))
				return
			}

			if txResult.Hash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				log.Info("transaction complete", "URL", "https://rinkeby.etherscan.io/tx/%s"+txResult.Hash.Hex())
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txID := range parsedIDs {
			s.False(backend.TransactionQueue().Has(status.QueuedTxID(txID)),
				"txqueue should not have test tx at this point (it should be completed)")
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
	case <-time.After(30 * time.Second):
		s.Fail("test timed out")
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		s.Fail("tx queue must be empty at this point")
		return
	}
}

func (s *BackendTestSuite) TestDiscardMultipleQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to sync

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIDs := make(chan string, testTxCount)
	allTestTxDiscarded := make(chan struct{}, 1)

	// replace transaction notification handler
	txFailedEventCallCount := 0
	node.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txID string
		var envelope node.SignalEnvelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)
		if envelope.Type == node.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			log.Info("transaction queued (will be discarded soon)", "id", txID)

			s.True(backend.TransactionQueue().Has(status.QueuedTxID(txID)), "txqueue should still have test tx")
			txIDs <- txID
		}

		if envelope.Type == node.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			log.Info("transaction return event received", "id", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := status.ErrQueuedTxDiscarded.Error()
			s.Equal(receivedErrMessage, expectedErrMessage)

			receivedErrCode := event["error_code"].(string)
			s.Equal("4", receivedErrCode)

			txFailedEventCallCount++
			if txFailedEventCallCount == testTxCount {
				allTestTxDiscarded <- struct{}{}
			}
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		s.EqualError(err, status.ErrQueuedTxDiscarded.Error())

		s.True(reflect.DeepEqual(txHashCheck, gethcommon.Hash{}), "transaction returned hash, while it shouldn't")
	}

	// wait for transactions, and discard immediately
	discardTxs := func(txIDStrings string) {
		var parsedIDs []string
		err := json.Unmarshal([]byte(txIDStrings), &parsedIDs)
		s.NoError(err)

		parsedIDs = append(parsedIDs, "invalid-tx-id")
		updatedTxIDStrings, _ := json.Marshal(parsedIDs)

		// discard
		discardResults := s.backend.DiscardTransactions(string(updatedTxIDStrings))
		if len(discardResults) != 1 || discardResults["invalid-tx-id"].Error.Error() != "transaction hash not found" {
			s.Fail(fmt.Sprintf("cannot discard txs: %v", discardResults))
			return
		}

		// try completing discarded transaction
		completeResults := s.backend.CompleteTransactions(string(updatedTxIDStrings), TestConfig.Account1.Password)
		if len(completeResults) != (testTxCount + 1) {
			s.Fail(fmt.Sprint("unexpected number of errors (call to CompleteTransaction should not succeed)"))
		}

		for _, txResult := range completeResults {
			if txResult.Error.Error() != "transaction hash not found" {
				s.Fail(fmt.Sprintf("invalid error for %s", txResult.Hash.Hex()))
				return
			}
			if txResult.Hash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				s.Fail(fmt.Sprintf("invalid hash (expected zero): %s", txResult.Hash.Hex()))
				return
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txID := range parsedIDs {
			if backend.TransactionQueue().Has(status.QueuedTxID(txID)) {
				s.Fail(fmt.Sprintf("txqueue should not have test tx at this point (it should be discarded): %s", txID))
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
	case <-time.After(30 * time.Second):
		s.Fail("test timed out")
		return
	}

	if backend.TransactionQueue().Count() != 0 {
		s.Fail("tx queue must be empty at this point")
		return
	}
}

func (s *BackendTestSuite) TestNonExistentQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	// replace transaction notification handler
	node.SetDefaultNodeNotificationHandler(func(string) {})

	// try completing non-existing transaction
	_, err := s.backend.CompleteTransaction("some-bad-transaction-id", TestConfig.Account1.Password)
	s.Error(err, "error expected and not received")
	s.EqualError(err, status.ErrQueuedTxIDNotFound.Error())
}

func (s *BackendTestSuite) TestEvictionOfQueuedTransactions() {
	require := s.Require()
	require.NotNil(s.backend)

	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	backend := s.LightEthereumService().StatusBackend
	require.NotNil(backend)

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	require.NoError(s.backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password))

	txQueue := backend.TransactionQueue()
	var i = 0
	txIDs := [status.DefaultTxQueueCap + 5 + 10]status.QueuedTxID{}
	backend.SetTransactionQueueHandler(func(queuedTx status.QueuedTx) {
		log.Info("tx enqueued", "i", i+1, "queue size", txQueue.Count(), "id", queuedTx.ID)
		txIDs[i] = queuedTx.ID
		i++
	})

	s.Zero(txQueue.Count(), "transaction count should be zero")

	for i := 0; i < 10; i++ {
		go backend.SendTransaction(nil, status.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(1 * time.Second)

	log.Info(fmt.Sprintf("Number of transactions queued: %d. Queue size (shouldn't be more than %d): %d",
		i, status.DefaultTxQueueCap, txQueue.Count()))

	s.Equal(10, txQueue.Count(), "transaction count should be 10")

	for i := 0; i < status.DefaultTxQueueCap+5; i++ { // stress test by hitting with lots of goroutines
		go backend.SendTransaction(nil, status.SendTxArgs{}) // nolint: errcheck
	}
	time.Sleep(3 * time.Second)

	if txQueue.Count() > status.DefaultTxQueueCap {
		s.Fail(fmt.Sprintf("transaction count should be %d (or %d): got %d", status.DefaultTxQueueCap, status.DefaultTxQueueCap-1, txQueue.Count()))
		return
	}

	for _, txID := range txIDs {
		txQueue.Remove(txID)
	}

	if txQueue.Count() != 0 {
		s.Fail(fmt.Sprintf("transaction count should be zero: %d", txQueue.Count()))
		return
	}
}
