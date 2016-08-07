package main

import (
	"fmt"
	"os"
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
	"math/big"
	"path/filepath"
)

const (
	testDataDir         = ".ethereumtest"
	testAddress         = "0x89b50b2b26947ccad43accaef76c21d175ad85f4"
	testAddressPassword = "asdf"
)

// TestAccountBindings makes sure we can create an account and subsequently
// unlock that account
func TestAccountBindings(t *testing.T) {
	rpcport = 8546 // in order to avoid conflicts with running react-native app

	dataDir, err := preprocessDataDir(testDataDir)
	if err != nil {
		glog.V(logger.Warn).Infoln("make node failed:", err)
	}

	// start geth node and wait for it to initialize
	go createAndStartNode(dataDir)
	time.Sleep(120 * time.Second) // LES syncs headers, so that we are up do date when it is done
	if currentNode == nil {
		t.Error("Test failed: could not start geth node")
	}

	// create an account
	address, pubkey, err := createAccount("badpassword")
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
	}

	// unlock the created account
	err = unlockAccount(address, "badpassword", 3)
	if err != nil {
		fmt.Println(err)
		t.Error("Test failed: could not unlock account")
	}
	time.Sleep(2 * time.Second)

	// test to see if the account was injected in whisper
	var whisperInstance *whisper.Whisper
	if err := currentNode.Service(&whisperInstance); err != nil {
		t.Errorf("whisper service not running: %v", err)
	}
	identitySucsess := whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubkey)))
	if !identitySucsess || err != nil {
		t.Errorf("Test failed: identity not injected into whisper: %v", err)
	}

	// test to see if we can post with the injected whisper identity
	postArgs := whisper.PostArgs{
		From:    pubkey,
		To:      pubkey,
		TTL:     100,
		Topics:  [][]byte{[]byte("test topic")},
		Payload: "test message",
	}
	whisperAPI := whisper.NewPublicWhisperAPI(whisperInstance)
	postSuccess, err := whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("Test failed: Could not post to whisper: %v", err)
	}

	// import test account (with test ether on it)
	err = copyFile(filepath.Join(testDataDir, "testnet", "keystore", "test-account.pk"), filepath.Join("data", "test-account.pk"))
	if err != nil {
		t.Errorf("Test failed: cannot copy test account PK: %v", err)
		return
	}
	time.Sleep(2 * time.Second)

	// unlock test account (to send ether from it)
	err = unlockAccount(testAddress, testAddressPassword, 300)
	if err != nil {
		fmt.Println(err)
		t.Error("Test failed: could not unlock account")
	}
	time.Sleep(2 * time.Second)

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
		if err := completeTransaction(queuedTx.Hash.Hex()); err != nil {
			t.Errorf("Test failed: cannot complete queued transation[%s]: %v", queuedTx.Hash.Hex(), err)
		}
		sentinel = 1
	})

	// try completing non-existing transaction
	if err := completeTransaction("0x1234512345123451234512345123456123451234512345123451234512345123"); err == nil {
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

	time.Sleep(10 * time.Second)
	if sentinel != 1 {
		t.Error("Test failed: transaction was never queued or completed")
	}

	// clean up
	err = os.RemoveAll(".ethereumtest")
	if err != nil {
		t.Error("Test failed: could not clean up temporary datadir")
	}
}
