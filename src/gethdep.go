package main

/*
#include <stddef.h>
#include <stdbool.h>
#include <jni.h>
extern bool GethServiceSignalEvent( const char *jsonEvent );
*/
import "C"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/p2p/discover"
	errextra "github.com/pkg/errors"
)

var (
	scryptN = 4096
	scryptP = 6
)

// createAccount creates an internal geth account
func createAccount(password string) (string, string, error) {

	if currentNode != nil {

		w := true

		if accountManager != nil {
			// generate the account
			account, err := accountManager.NewAccount(password, w)
			if err != nil {
				return "", "", errextra.Wrap(err, "Account manager could not create the account")
			}
			address := fmt.Sprintf("%x", account.Address)

			// recover the public key to return
			keyContents, err := ioutil.ReadFile(account.File)
			if err != nil {
				return address, "", errextra.Wrap(err, "Could not load the key contents")
			}
			key, err := accounts.DecryptKey(keyContents, password)
			if err != nil {
				return address, "", errextra.Wrap(err, "Could not recover the key")
			}
			pubKey := common.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

			return address, pubKey, nil
		}
		return "", "", errors.New("Could not retrieve account manager")

	}

	return "", "", errors.New("No running node detected for account creation")

}

// unlockAccount unlocks an existing account for a certain duration and
// inject the account as a whisper identity if the account was created as
// a whisper enabled account
func unlockAccount(address, password string, seconds int) error {

	if currentNode != nil {

		if accountManager != nil {
			account, err := utils.MakeAddress(accountManager, address)
			if err != nil {
				return errextra.Wrap(err, "Could not retrieve account from address")
			}

			err = accountManager.TimedUnlock(account, password, time.Duration(seconds)*time.Second)
			if err != nil {
				return errextra.Wrap(err, "Could not decrypt account")
			}
			return nil
		}
		return errors.New("Could not retrieve account manager")
	}

	return errors.New("No running node detected for account unlock")

}

// createAndStartNode creates a node entity and starts the
// node running locally
func createAndStartNode(inputDir string) error {

	currentNode = MakeNode(inputDir)
	if currentNode != nil {
		RunNode(currentNode)
		return nil
	}

	return errors.New("Could not create the in-memory node object")

}

func doAddPeer(url string) (bool, error) {
	server := currentNode.Server()
	if server == nil {
		return false, errors.New("node not started")
	}
	// Try to add the url as a static peer and return
	node, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddPeer(node)
	return true, nil
}

func onSendTransactionRequest(queuedTx les.QueuedTx) {
	event := GethEvent{
		Type: "sendTransactionQueued",
		Event: SendTransactionEvent{
			Hash: queuedTx.Hash.Hex(),
			Args: queuedTx.Args,
		},
	}

	body, _ := json.Marshal(&event)
	C.GethServiceSignalEvent(C.CString(string(body)))
}

func completeTransaction(hash string) (common.Hash, error) {
	if currentNode != nil {
		if lightEthereum != nil {
			backend := lightEthereum.StatusBackend

			return backend.CompleteQueuedTransaction(les.QueuedTxHash(hash))
		}

		return common.Hash{}, errors.New("can not retrieve LES service")
	}

	return common.Hash{}, errors.New("can not complete transaction: no running node detected")
}
