package main

import "C"
import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth"
)

const (
	testDataDir         = "../../.ethereumtest"
	testNodeSyncSeconds = 120
	testAddress         = "0xadaf150b905cf5e6a778e553e15a139b6618bbb7"
	testAddressPassword = "asdfasdf"
	newAccountPassword  = "badpassword"
	testAddress1        = "0xadd4d1d02e71c7360c53296968e59d57fd15e2ba"
	testStatusJsFile    = "../../jail/testdata/status.js"
)

func testExportedAPI(t *testing.T, done chan struct{}) {
	<-startTestNode(t)

	tests := []struct {
		name string
		fn   func(t *testing.T) bool
	}{
		{
			"restart node RPC",
			testRestartNodeRPC,
		},
		{
			"create main and child accounts",
			testCreateChildAccount,
		},
		{
			"recover account",
			testRecoverAccount,
		},
		{
			"account select/login",
			testAccountSelect,
		},
		{
			"account logout",
			testAccountLogout,
		},
		{
			"complete single queued transaction",
			testCompleteTransaction,
		},
		{
			"test complete multiple queued transactions",
			testCompleteMultipleQueuedTransactions,
		},
		{
			"discard single queued transaction",
			testDiscardTransaction,
		},
		{
			"test discard multiple queued transactions",
			testDiscardMultipleQueuedTransactions,
		},
		{
			"test jail initialization",
			testJailInit,
		},
		{
			"test jailed calls",
			testJailFunctionCall,
		},
	}

	for _, test := range tests {
		if ok := test.fn(t); !ok {
			break
		}
	}

	done <- struct{}{}
}

func testRestartNodeRPC(t *testing.T) bool {
	// stop RPC
	stopNodeRPCServerResponse := geth.JSONError{}
	rawResponse := StopNodeRPCServer()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &stopNodeRPCServerResponse); err != nil {
		t.Errorf("cannot decode StopNodeRPCServer reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}
	if stopNodeRPCServerResponse.Error != "" {
		t.Errorf("unexpected error: %s", stopNodeRPCServerResponse.Error)
		return false
	}

	// start again RPC
	startNodeRPCServerResponse := geth.JSONError{}
	rawResponse = StartNodeRPCServer()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &startNodeRPCServerResponse); err != nil {
		t.Errorf("cannot decode StartNodeRPCServer reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}
	if startNodeRPCServerResponse.Error != "" {
		t.Errorf("unexpected error: %s", startNodeRPCServerResponse.Error)
		return false
	}

	// start when we have RPC already running
	startNodeRPCServerResponse = geth.JSONError{}
	rawResponse = StartNodeRPCServer()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &startNodeRPCServerResponse); err != nil {
		t.Errorf("cannot decode StartNodeRPCServer reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}
	expectedError := "HTTP RPC already running on localhost:8545"
	if startNodeRPCServerResponse.Error != expectedError {
		t.Errorf("expected error not thrown: %s", expectedError)
		return false
	}

	return true
}

func testCreateChildAccount(t *testing.T) bool {
	geth.Logout() // to make sure that we start with empty account (which might get populated during previous tests)

	accountManager, err := geth.NodeManagerInstance().AccountManager()
	if err != nil {
		t.Error(err)
		return false
	}

	// create an account
	createAccountResponse := geth.AccountInfo{}
	rawResponse := CreateAccount(C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createAccountResponse); err != nil {
		t.Errorf("cannot decode CreateAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createAccountResponse.Error != "" {
		t.Errorf("could not create account: %s", err)
		return false
	}
	address, pubKey, mnemonic := createAccountResponse.Address, createAccountResponse.PubKey, createAccountResponse.Mnemonic
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	account, err := geth.ParseAccountString(accountManager, address)
	if err != nil {
		t.Errorf("can not get account from address: %v", err)
		return false
	}

	// obtain decrypted key, and make sure that extended key (which will be used as root for sub-accounts) is present
	account, key, err := accountManager.AccountDecryptedKey(account, newAccountPassword)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return false
	}

	if key.ExtendedKey == nil {
		t.Error("CKD#2 has not been generated for new account")
		return false
	}

	// try creating sub-account, w/o selecting main account i.e. w/o login to main account
	createSubAccountResponse := geth.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse); err != nil {
		t.Errorf("cannot decode CreateChildAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse.Error != geth.ErrNoAccountSelected.Error() {
		t.Errorf("expected error is not returned (tried to create sub-account w/o login): %v", createSubAccountResponse.Error)
		return false
	}

	err = geth.SelectAccount(address, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}

	// try to create sub-account with wrong password
	createSubAccountResponse = geth.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString("wrong password"))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse); err != nil {
		t.Errorf("cannot decode CreateChildAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse.Error != "cannot retreive a valid key for a given account: could not decrypt key with given passphrase" {
		t.Errorf("expected error is not returned (tried to create sub-account with wrong password): %v", createSubAccountResponse.Error)
		return false
	}

	// create sub-account (from implicit parent)
	createSubAccountResponse1 := geth.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse1); err != nil {
		t.Errorf("cannot decode CreateChildAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse1.Error != "" {
		t.Errorf("cannot create sub-account: %v", createSubAccountResponse1.Error)
		return false
	}

	// make sure that sub-account index automatically progresses
	createSubAccountResponse2 := geth.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse2); err != nil {
		t.Errorf("cannot decode CreateChildAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse2.Error != "" {
		t.Errorf("cannot create sub-account: %v", createSubAccountResponse2.Error)
	}

	if createSubAccountResponse1.Address == createSubAccountResponse2.Address || createSubAccountResponse1.PubKey == createSubAccountResponse2.PubKey {
		t.Error("sub-account index auto-increament failed")
		return false
	}

	// create sub-account (from explicit parent)
	createSubAccountResponse3 := geth.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(createSubAccountResponse2.Address), C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse3); err != nil {
		t.Errorf("cannot decode CreateChildAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse3.Error != "" {
		t.Errorf("cannot create sub-account: %v", createSubAccountResponse3.Error)
	}

	subAccount1, subAccount2, subAccount3 := createSubAccountResponse1.Address, createSubAccountResponse2.Address, createSubAccountResponse3.Address
	subPubKey1, subPubKey2, subPubKey3 := createSubAccountResponse1.PubKey, createSubAccountResponse2.PubKey, createSubAccountResponse3.PubKey

	if subAccount1 == subAccount3 || subPubKey1 == subPubKey3 || subAccount2 == subAccount3 || subPubKey2 == subPubKey3 {
		t.Error("sub-account index auto-increament failed")
		return false
	}

	return true
}

func testRecoverAccount(t *testing.T) bool {
	accountManager, _ := geth.NodeManagerInstance().AccountManager()

	// create an account
	address, pubKey, mnemonic, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// try recovering using password + mnemonic
	recoverAccountResponse := geth.AccountInfo{}
	rawResponse := RecoverAccount(C.CString(newAccountPassword), C.CString(mnemonic))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed: %v", recoverAccountResponse.Error)
		return false
	}
	addressCheck, pubKeyCheck := recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details")
	}

	// now test recovering, but make sure that account/key file is removed i.e. simulate recovering on a new device
	account, err := geth.ParseAccountString(accountManager, address)
	if err != nil {
		t.Errorf("can not get account from address: %v", err)
	}

	account, key, err := accountManager.AccountDecryptedKey(account, newAccountPassword)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return false
	}
	extChild2String := key.ExtendedKey.String()

	if err := accountManager.DeleteAccount(account, newAccountPassword); err != nil {
		t.Errorf("cannot remove account: %v", err)
	}

	recoverAccountResponse = geth.AccountInfo{}
	rawResponse = RecoverAccount(C.CString(newAccountPassword), C.CString(mnemonic))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed (for non-cached account): %v", recoverAccountResponse.Error)
		return false
	}
	addressCheck, pubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details (for non-cached account)")
	}

	// make sure that extended key exists and is imported ok too
	account, key, err = accountManager.AccountDecryptedKey(account, newAccountPassword)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return false
	}
	if extChild2String != key.ExtendedKey.String() {
		t.Errorf("CKD#2 key mismatch, expected: %s, got: %s", extChild2String, key.ExtendedKey.String())
	}

	// make sure that calling import several times, just returns from cache (no error is expected)
	recoverAccountResponse = geth.AccountInfo{}
	rawResponse = RecoverAccount(C.CString(newAccountPassword), C.CString(mnemonic))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed (for non-cached account): %v", recoverAccountResponse.Error)
		return false
	}
	addressCheck, pubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details (for non-cached account)")
	}

	// time to login with recovered data
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// make sure that identity is not (yet injected)
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKeyCheck))) {
		t.Error("identity already present in whisper")
	}
	err = geth.SelectAccount(addressCheck, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKeyCheck))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	return true
}

func testAccountSelect(t *testing.T) bool {
	// test to see if the account was injected in whisper
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	address1, pubKey1, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Error("Test failed: could not create account")
		return false
	}
	t.Logf("Account created: {address: %s, key: %s}", address2, pubKey2)

	// make sure that identity is not (yet injected)
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity already present in whisper")
	}

	// try selecting with wrong password
	loginResponse := geth.JSONError{}
	rawResponse := Login(C.CString(address1), C.CString("wrongPassword"))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error == "" {
		t.Error("select account is expected to throw error: wrong password used")
		return false
	}

	loginResponse = geth.JSONError{}
	rawResponse = Login(C.CString(address1), C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// select another account, make sure that previous account is wiped out from Whisper cache
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Error("identity already present in whisper")
	}

	loginResponse = geth.JSONError{}
	rawResponse = Login(C.CString(address2), C.CString(newAccountPassword))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not select account: %v", loginResponse.Error)
		return false
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity should be removed, but it is still present in whisper")
	}

	return true
}

func testAccountLogout(t *testing.T) bool {
	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
		return false
	}

	// create an account
	address, pubKey, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}

	// make sure that identity doesn't exist (yet) in Whisper
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey))) {
		t.Error("identity already present in whisper")
		return false
	}

	// select/login
	err = geth.SelectAccount(address, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey))) {
		t.Error("identity not injected into whisper")
		return false
	}

	logoutResponse := geth.JSONError{}
	rawResponse := Logout()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &logoutResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if logoutResponse.Error != "" {
		t.Errorf("cannot logout: %v", logoutResponse.Error)
		return false
	}

	// now, logout and check if identity is removed indeed
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey))) {
		t.Error("identity not cleared from whisper")
		return false
	}

	return true
}

func testCompleteTransaction(t *testing.T) bool {
	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return false
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testAddress, testAddressPassword); err != nil {
		t.Errorf("cannot select account: %v", testAddress)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	queuedTxCompleted := make(chan struct{}, 1)
	abortPanic := make(chan struct{}, 1)
	geth.PanicAfter(10*time.Second, abortPanic, "testCompleteTransaction")

	// replace transaction notification handler
	var txHash = ""
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction queued (will be completed in 5 secs): {id: %s}\n", event["id"].(string))
			time.Sleep(5 * time.Second)

			completeTxResponse := geth.CompleteTransactionResult{}
			rawResponse := CompleteTransaction(C.CString(event["id"].(string)), C.CString(testAddressPassword))

			if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &completeTxResponse); err != nil {
				t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
			}

			if completeTxResponse.Error != "" {
				t.Errorf("cannot complete queued transation[%v]: %v", event["id"], completeTxResponse.Error)
			}

			txHash = completeTxResponse.Hash

			t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txHash)
			abortPanic <- struct{}{} // so that timeout is aborted
			queuedTxCompleted <- struct{}{}
		}
	})

	//  this call blocks, up until Complete Transaction is called
	txHashCheck, err := backend.SendTransaction(nil, status.SendTxArgs{
		From:  geth.FromAddress(testAddress),
		To:    geth.ToAddress(testAddress1),
		Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("Test failed: cannot send transaction: %v", err)
	}

	<-queuedTxCompleted // make sure that complete transaction handler completes its magic, before we proceed

	if txHash != txHashCheck.Hex() {
		t.Errorf("Transaction hash returned from SendTransaction is invalid: expected %s, got %s", txHashCheck.Hex(), txHash)
		return false
	}

	if reflect.DeepEqual(txHashCheck, common.Hash{}) {
		t.Error("Test failed: transaction was never queued or completed")
		return false
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	return true
}

func testCompleteMultipleQueuedTransactions(t *testing.T) bool {
	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return false
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testAddress, testAddressPassword); err != nil {
		t.Errorf("cannot select account: %v", testAddress)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIds := make(chan string, testTxCount)
	allTestTxCompleted := make(chan struct{}, 1)

	// replace transaction notification handler
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txId string
		var envelope geth.SignalEnvelope
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

func testDiscardTransaction(t *testing.T) bool {
	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return false
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testAddress, testAddressPassword); err != nil {
		t.Errorf("cannot select account: %v", testAddress)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	geth.PanicAfter(20*time.Second, completeQueuedTransaction, "TestDiscardQueuedTransactions")

	// replace transaction notification handler
	var txId string
	txFailedEventCalled := false
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope geth.SignalEnvelope
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

			// discard
			discardResponse := geth.DiscardTransactionResult{}
			rawResponse := DiscardTransaction(C.CString(txId))

			if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &discardResponse); err != nil {
				t.Errorf("cannot decode RecoverAccount reponse (%s): %v", C.GoString(rawResponse), err)
			}

			if discardResponse.Error != "" {
				t.Errorf("cannot discard tx: %v", discardResponse.Error)
				return
			}

			// try completing discarded transaction
			_, err = geth.CompleteTransaction(txId, testAddressPassword)
			if err.Error() != "transaction hash not found" {
				t.Error("expects tx not found, but call to CompleteTransaction succeeded")
				return
			}

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			if backend.TransactionQueue().Has(status.QueuedTxId(txId)) {
				t.Errorf("txqueue should not have test tx at this point (it should be discarded): %s", txId)
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
		From:  geth.FromAddress(testAddress),
		To:    geth.ToAddress(testAddress1),
		Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
	})
	if err != status.ErrQueuedTxDiscarded {
		t.Errorf("expected error not thrown: %v", err)
		return false
	}

	if !reflect.DeepEqual(txHashCheck, common.Hash{}) {
		t.Error("transaction returned hash, while it shouldn't")
		return false
	}

	if backend.TransactionQueue().Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	if !txFailedEventCalled {
		t.Error("expected tx failure signal is not received")
		return false
	}

	return true
}

func testDiscardMultipleQueuedTransactions(t *testing.T) bool {
	// obtain reference to status backend
	lightEthereum, err := geth.NodeManagerInstance().LightEthereumService()
	if err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
		return false
	}
	backend := lightEthereum.StatusBackend

	// reset queue
	backend.TransactionQueue().Reset()

	// log into account from which transactions will be sent
	if err := geth.SelectAccount(testAddress, testAddressPassword); err != nil {
		t.Errorf("cannot select account: %v", testAddress)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIds := make(chan string, testTxCount)
	allTestTxDiscarded := make(chan struct{}, 1)

	// replace transaction notification handler
	txFailedEventCallCount := 0
	geth.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txId string
		var envelope geth.SignalEnvelope
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

func testJailInit(t *testing.T) bool {
	initCode := `
	var _status_catalog = {
		foo: 'bar'
	};
	`
	InitJail(C.CString(initCode))

	extraCode := `
	var extraFunc = function (x) {
	  return x * x;
	};
	`
	rawResponse := Parse(C.CString("CHAT_ID_INIT_TEST"), C.CString(extraCode))
	parsedResponse := C.GoString(rawResponse)

	expectedResponse := `{"result": {"foo":"bar"}}`

	if !reflect.DeepEqual(expectedResponse, parsedResponse) {
		t.Error("expected output not returned from jail.Parse()")
		return false
	}

	t.Logf("jail inited and parsed: %s", parsedResponse)

	return true
}

func testJailFunctionCall(t *testing.T) bool {
	InitJail(C.CString(""))

	// load Status JS and add test command to it
	statusJS := geth.LoadFromFile(testStatusJsFile) + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	Parse(C.CString("CHAT_ID_CALL_TEST"), C.CString(statusJS))

	// call with wrong chat id
	rawResponse := Call(C.CString("CHAT_IDNON_EXISTENT"), C.CString(""), C.CString(""))
	parsedResponse := C.GoString(rawResponse)
	expectedError := `{"error":"Cell[CHAT_IDNON_EXISTENT] doesn't exist."}`
	if parsedResponse != expectedError {
		t.Errorf("expected error is not returned: expected %s, got %s", expectedError, parsedResponse)
		return false
	}

	// call extraFunc()
	rawResponse = Call(C.CString("CHAT_ID_CALL_TEST"), C.CString(`["commands", "testCommand"]`), C.CString(`{"val": 12}`))
	parsedResponse = C.GoString(rawResponse)
	expectedResponse := `{"result": 144}`
	if parsedResponse != expectedResponse {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, parsedResponse)
		return false
	}

	t.Logf("jailed method called: %s", parsedResponse)

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
		var envelope geth.SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == geth.EventNodeCrashed {
			geth.TriggerDefaultNodeNotificationHandler(jsonEvent)
			return
		}

		if envelope.Type == geth.EventTransactionQueued {
		}
		if envelope.Type == geth.EventNodeStarted {
			// manually add static nodes (LES auto-discovery is not stable yet)
			configFile, err := ioutil.ReadFile(filepath.Join("../../data", "static-nodes.json"))
			if err != nil {
				return
			}
			var enodes []string
			if err = json.Unmarshal(configFile, &enodes); err != nil {
				return
			}

			for _, enode := range enodes {
				AddPeer(C.CString(enode))
			}

			// sync
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

	go func() {
		response := StartNode(C.CString(testDataDir))
		err := geth.JSONError{}

		json.Unmarshal([]byte(C.GoString(response)), &err)
		if err.Error != "" {
			t.Error("cannot start node")
		}
	}()

	return waitForNodeStart
}
