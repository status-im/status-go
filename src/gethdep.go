package main

/*
#include <stddef.h>
#include <stdbool.h>
#include <jni.h>
extern bool GethServiceSignalEvent( const char *jsonEvent );
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/p2p/discover"
	errextra "github.com/pkg/errors"
	"github.com/status-im/status-go/src/extkeys"
)

var (
	ErrInvalidGethNode                 = errors.New("no running node detected for account unlock")
	ErrInvalidWhisperService           = errors.New("whisper service is unavailable")
	ErrInvalidAccountManager           = errors.New("could not retrieve account manager")
	ErrAddressToAccountMappingFailure  = errors.New("cannot retreive a valid account for a given address")
	ErrAccountToKeyMappingFailure      = errors.New("cannot retreive a valid key for a given account")
	ErrUnlockCalled                    = errors.New("no need to unlock accounts, use Login() instead")
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	ErrWhisperClearIdentitiesFailure   = errors.New("failed to clear whisper identities")
)

// createAccount creates an internal geth account
func createAccount(password string) (string, string, string, error) {

	if currentNode != nil {

		w := true

		if accountManager != nil {
			// generate mnemonic phrase
			m := extkeys.NewMnemonic()
			mnemonic, err := m.MnemonicPhrase(128, extkeys.EnglishLanguage)
			if err != nil {
				return "", "", "", errextra.Wrap(err, "Can not create mnemonic seed")
			}

			// generate extended master key (see BIP32)
			extKey, err := extkeys.NewMaster(m.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
			if err != nil {
				return "", "", "", errextra.Wrap(err, "Can not create master extended key")
			}

			// derive hardened child (see BIP44)
			extChild1, err := extKey.BIP44Child(extkeys.CoinTypeETH, 0)
			if err != nil {
				return "", "", "", errextra.Wrap(err, "Can not derive hardened child key (#1)")
			}

			// generate the account
			account, err := accountManager.NewAccountUsingExtendedKey(extChild1, password, w)
			if err != nil {
				return "", "", "", errextra.Wrap(err, "Account manager could not create the account")
			}
			address := fmt.Sprintf("%x", account.Address)

			// recover the public key to return
			account, key, err := accountManager.AccountDecryptedKey(account, password)
			if err != nil {
				return address, "", "", errextra.Wrap(err, "Could not recover the key")
			}
			pubKey := common.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

			return address, pubKey, mnemonic, nil
		}

		return "", "", "", errors.New("Could not retrieve account manager")
	}

	return "", "", "", errors.New("No running node detected for account creation")
}

func recoverAccount(password, mnemonic string) (string, string, error) {

	if currentNode != nil {

		if accountManager != nil {
			m := extkeys.NewMnemonic()
			// re-create extended key (see BIP32)
			extKey, err := extkeys.NewMaster(m.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
			if err != nil {
				return "", "", errextra.Wrap(err, "Can not create master extended key")
			}

			// derive hardened child (see BIP44)
			extChild1, err := extKey.BIP44Child(extkeys.CoinTypeETH, 0)
			if err != nil {
				return "", "", errextra.Wrap(err, "Can not derive hardened child key (#1)")
			}

			privateKeyECDSA := extChild1.ToECDSA()
			address := fmt.Sprintf("%x", crypto.PubkeyToAddress(privateKeyECDSA.PublicKey))
			pubKey := common.ToHex(crypto.FromECDSAPub(&privateKeyECDSA.PublicKey))

			accountManager.ImportECDSA(privateKeyECDSA, password) // try caching key, ignore errors

			return address, pubKey, nil
		}

		return "", "", errors.New("Could not retrieve account manager")
	}

	return "", "", errors.New("No running node detected for account unlock")
}

func selectAccount(address, password string) error {
	if currentNode == nil {
		return ErrInvalidGethNode
	}

	if accountManager == nil {
		return ErrInvalidAccountManager
	}
	account, err := utils.MakeAddress(accountManager, address)
	if err != nil {
		return ErrAddressToAccountMappingFailure
	}

	account, accountKey, err := accountManager.AccountDecryptedKey(account, password)
	if err != nil {
		return fmt.Errorf("%s: %v", ErrAccountToKeyMappingFailure.Error(), err)
	}

	if whisperService == nil {
		return ErrInvalidWhisperService
	}
	if err := whisperService.InjectIdentity(accountKey.PrivateKey); err != nil {
		return ErrWhisperIdentityInjectionFailure
	}

	return nil
}

// logout clears whisper identities
func logout() error {
	if currentNode == nil {
		return ErrInvalidGethNode
	}
	if whisperService == nil {
		return ErrInvalidWhisperService
	}

	err := whisperService.ClearIdentities()
	if err != nil {
		return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
	}

	return nil
}

// unlockAccount unlocks an existing account for a certain duration and
// inject the account as a whisper identity if the account was created as
// a whisper enabled account
func unlockAccount(address, password string, seconds int) error {
	if currentNode == nil {
		return ErrInvalidGethNode
	}

	return ErrUnlockCalled
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

func onSendTransactionRequest(queuedTx status.QueuedTx) {
	event := GethEvent{
		Type: "sendTransactionQueued",
		Event: SendTransactionEvent{
			Id:   string(queuedTx.Id),
			Args: queuedTx.Args,
		},
	}

	body, _ := json.Marshal(&event)
	C.GethServiceSignalEvent(C.CString(string(body)))
}

func completeTransaction(id, password string) (common.Hash, error) {
	if currentNode != nil {
		if lightEthereum != nil {
			backend := lightEthereum.StatusBackend

			return backend.CompleteQueuedTransaction(status.QueuedTxId(id), password)
		}

		return common.Hash{}, errors.New("can not retrieve LES service")
	}

	return common.Hash{}, errors.New("can not complete transaction: no running node detected")
}
