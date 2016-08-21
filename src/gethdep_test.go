package main

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper"
)

const (
	testDataDir         = ".ethereumtest"
	testAddress         = "0x89b50b2b26947ccad43accaef76c21d175ad85f4"
	testAddressPassword = "asdf"
	testNodeSyncSeconds = 180
	newAccountPassword  = "badpassword"

	whisperMessage1 = "test message 1 (K1 -> K1)"
	whisperMessage2 = "test message 2 (K1 -> '')"
	whisperMessage3 = "test message 3 ('' -> '')"
	whisperMessage4 = "test message 4 ('' -> K1)"
	whisperMessage5 = "test message 5 (K2 -> K1)"
)

func TestRemindAccountDetails(t *testing.T) {
	err := prepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// create an account
	address, pubKey, mnemonic, err := createAccount(newAccountPassword)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return
	}
	glog.V(logger.Info).Infof("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// try reminding using password + mnemonic
	addressCheck, pubKeyCheck, err := remindAccountDetails(newAccountPassword, mnemonic)
	if err != nil {
		t.Errorf("remind details failed: %v", err)
		return
	}
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("Test failed: remind account details failed to pull the correct details")
	}
}

func TestAccountSelect(t *testing.T) {

	err := prepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// test to see if the account was injected in whisper
	var whisperInstance *whisper.Whisper
	if err := currentNode.Service(&whisperInstance); err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an accounts
	address1, pubKey1, _, err := createAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	glog.V(logger.Info).Infof("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := createAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	glog.V(logger.Info).Infof("Account created: {address: %s, key: %s}", address2, pubKey2)

	// inject key of newly created account into Whisper, as identity
	if whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity already present in whisper")
	}

	// try selecting with wrong password
	err = selectAccount(address1, "wrongPassword")
	if err == nil {
		t.Errorf("select account is expected to throw error: wrong password used")
		return
	}
	err = selectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// select another account, make sure that previous account is wiped out from Whisper cache
	if whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Errorf("identity already present in whisper")
	}
	err = selectAccount(address2, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	if !whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity should be removed, but it is still present in whisper")
	}
}

func TestWhisperMessaging(t *testing.T) {
	err := prepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// test to see if the account was injected in whisper
	var whisperInstance *whisper.Whisper
	if err := currentNode.Service(&whisperInstance); err != nil {
		t.Errorf("whisper service not running: %v", err)
	}
	whisperAPI := whisper.NewPublicWhisperAPI(whisperInstance)

	// prepare message
	postArgs := whisper.PostArgs{
		From:    "",
		To:      "",
		TTL:     10,
		Topics:  [][]byte{[]byte("test topic")},
		Payload: "test message",
	}

	// create an accounts
	address1, pubKey1, _, err := createAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	glog.V(logger.Info).Infof("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := createAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	glog.V(logger.Info).Infof("Account created: {address: %s, key: %s}", address2, pubKey2)

	// start watchers
	var receivedMessages = map[string]bool{
		whisperMessage1: false,
		whisperMessage2: false,
		whisperMessage3: false,
		whisperMessage4: false,
		whisperMessage5: false,
	}
	whisperService.Watch(whisper.Filter{
		//From: crypto.ToECDSAPub(common.FromHex(pubKey1)),
		//To:   crypto.ToECDSAPub(common.FromHex(pubKey2)),
		Fn: func(msg *whisper.Message) {
			glog.V(logger.Info).Infof("Whisper message received: %s", msg.Payload)
			receivedMessages[string(msg.Payload)] = true
		},
	})

	// inject key of newly created account into Whisper, as identity
	if whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Errorf("identity already present in whisper")
	}
	err = selectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	identitySucceess := whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1)))
	if !identitySucceess || err != nil {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) { // ensure that second id is not injected
		t.Errorf("identity already present in whisper")
	}

	// double selecting (shouldn't be a problem)
	err = selectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}

	// TEST 0: From != nil && To != nil: encrypted signed message (but we cannot decrypt it - so watchers will not report this)
	postArgs.From = pubKey1
	postArgs.To = pubKey2 // owner of that public key will be able to decrypt it
	postSuccess, err := whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 1: From != nil && To != nil: encrypted signed message (to self)
	postArgs.From = pubKey1
	postArgs.To = pubKey1
	postArgs.Payload = whisperMessage1
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// send from account that is not in Whisper identity list
	postArgs.From = pubKey2
	postSuccess, err = whisperAPI.Post(postArgs)
	if err == nil || err.Error() != fmt.Sprintf("unknown identity to send from: %s", pubKey2) {
		t.Errorf("expected error not voiced: we are sending from non-injected whisper identity")
	}

	// TEST 2: From != nil && To == nil: signed broadcast (known sender)
	postArgs.From = pubKey1
	postArgs.To = ""
	postArgs.Payload = whisperMessage2
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 3: From == nil && To == nil: anonymous broadcast
	postArgs.From = ""
	postArgs.To = ""
	postArgs.Payload = whisperMessage3
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 4: From == nil && To != nil: encrypted anonymous message
	postArgs.From = ""
	postArgs.To = pubKey1
	postArgs.Payload = whisperMessage4
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 5: From != nil && To != nil: encrypted and signed response
	postArgs.From = ""
	postArgs.To = pubKey1
	postArgs.Payload = whisperMessage5
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	time.Sleep(2 * time.Second) // allow whisper to poll
	for message, status := range receivedMessages {
		if !status {
			t.Errorf("Expected message not received: %s", message)
		}
	}

}

func TestQueuedTransactions(t *testing.T) {
	err := prepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	// create an account
	address, pubKey, mnemonic, err := createAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	glog.V(logger.Info).Infof("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// test transaction queueing
	var lightEthereum *les.LightEthereum
	if err := currentNode.Service(&lightEthereum); err != nil {
		t.Errorf("Test failed: LES service is not running: %v", err)
	}
	backend := lightEthereum.StatusBackend

	// replace transaction notification hanlder
	sentinel := 0
	backend.SetTransactionQueueHandler(func(queuedTx les.QueuedTx) {
		glog.V(logger.Info).Infof("Queued transaction hash: %v\n", queuedTx.Hash.Hex())
		var txHash common.Hash
		if txHash, err = completeTransaction(queuedTx.Hash.Hex(), testAddressPassword); err != nil {
			t.Errorf("Test failed: cannot complete queued transation[%s]: %v", queuedTx.Hash.Hex(), err)
			return
		}

		glog.V(logger.Info).Infof("Transaction complete: https://testnet.etherscan.io/tx/%s", txHash.Hex())
		sentinel = 1
	})

	// try completing non-existing transaction
	if _, err := completeTransaction("0x1234512345123451234512345123456123451234512345123451234512345123", testAddressPassword); err == nil {
		t.Errorf("Test failed: error expected and not recieved")
	}

	// send normal transaction
	from, err := utils.MakeAddress(accountManager, testAddress)
	if err != nil {
		t.Errorf("Test failed: Could not retrieve account from address: %v", err)
	}

	to, err := utils.MakeAddress(accountManager, address)
	if err != nil {
		t.Errorf("Test failed: Could not retrieve account from address: %v", err)
	}

	err = backend.SendTransaction(nil, les.SendTxArgs{
		From:  from.Address,
		To:    &to.Address,
		Value: rpc.NewHexNumber(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("Test failed: cannot send transaction: %v", err)
	}

	time.Sleep(15 * time.Second)
	if sentinel != 1 {
		t.Error("Test failed: transaction was never queued or completed")
	}

}

func prepareTestNode() error {
	if currentNode != nil {
		return nil
	}

	rpcport = 8546 // in order to avoid conflicts with running react-native app

	syncRequired := false
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		syncRequired = true
	}

	dataDir, err := preprocessDataDir(testDataDir)
	if err != nil {
		glog.V(logger.Warn).Infoln("make node failed:", err)
		return err
	}

	// import test account (with test ether on it)
	err = copyFile(filepath.Join(testDataDir, "testnet", "keystore", "test-account.pk"), filepath.Join("data", "test-account.pk"))
	if err != nil {
		glog.V(logger.Warn).Infof("Test failed: cannot copy test account PK: %v", err)
		return err
	}

	// start geth node and wait for it to initialize
	go createAndStartNode(dataDir)
	time.Sleep(5 * time.Second)
	if currentNode == nil {
		return errors.New("Test failed: could not start geth node")
	}

	if syncRequired {
		glog.V(logger.Warn).Infof("Sync is required, it will take %d seconds", testNodeSyncSeconds)
		time.Sleep(testNodeSyncSeconds * time.Second) // LES syncs headers, so that we are up do date when it is done
	} else {
		time.Sleep(10 * time.Second)
	}

	return nil
}

func cleanup() {
	err := os.RemoveAll(testDataDir)
	if err != nil {
		glog.V(logger.Warn).Infof("Test failed: could not clean up temporary datadir")
	}
}
