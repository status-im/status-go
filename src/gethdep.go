package main

/*
#include <stddef.h>
#include <stdbool.h>
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
	ErrUnlockCalled                    = errors.New("no need to unlock accounts, login instead")
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	ErrWhisperClearIdentitiesFailure   = errors.New("failed to clear whisper identities")
	ErrWhisperNoIdentityFound          = errors.New("failed to locate identity previously injected into Whisper")
	ErrNoAccountSelected               = errors.New("no account has been selected, please login")
)

// createAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func createAccount(password string) (address, pubKey, mnemonic string, err error) {
	if currentNode == nil {
		return "", "", "", ErrInvalidGethNode
	}

	if accountManager == nil {
		return "", "", "", ErrInvalidAccountManager
	}

	// generate mnemonic phrase
	m := extkeys.NewMnemonic()
	mnemonic, err = m.MnemonicPhrase(128, extkeys.EnglishLanguage)
	if err != nil {
		return "", "", "", errextra.Wrap(err, "Can not create mnemonic seed")
	}

	// generate extended master key (see BIP32)
	extKey, err := extkeys.NewMaster(m.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
	if err != nil {
		return "", "", "", errextra.Wrap(err, "Can not create master extended key")
	}

	// import created key into account keystore
	address, pubKey, err = importExtendedKey(extKey, password)
	if err != nil {
		return "", "", "", err
	}

	return address, pubKey, mnemonic, nil
}

// createChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func createChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	if currentNode == nil {
		return "", "", ErrInvalidGethNode
	}

	if accountManager == nil {
		return "", "", ErrInvalidAccountManager
	}

	if parentAddress == "" { // by default derive from currently selected account
		parentAddress = selectedAddress
	}

	if parentAddress == "" {
		return "", "", ErrNoAccountSelected
	}

	// make sure that given password can decrypt key associated with a given parent address
	account, err := utils.MakeAddress(accountManager, parentAddress)
	if err != nil {
		return "", "", ErrAddressToAccountMappingFailure
	}

	account, accountKey, err := accountManager.AccountDecryptedKey(account, password)
	if err != nil {
		return "", "", fmt.Errorf("%s: %v", ErrAccountToKeyMappingFailure.Error(), err)
	}

	parentKey, err := extkeys.NewKeyFromString(accountKey.ExtendedKey.String())
	if err != nil {
		return "", "", err
	}

	// derive child key
	childKey, err := parentKey.Child(accountKey.SubAccountIndex)
	if err != nil {
		return "", "", err
	}
	accountManager.IncSubAccountIndex(account, password)

	// import derived key into account keystore
	address, pubKey, err = importExtendedKey(childKey, password)
	if err != nil {
		return
	}

	return address, pubKey, nil
}

// recoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func recoverAccount(password, mnemonic string) (address, pubKey string, err error) {
	if currentNode == nil {
		return "", "", ErrInvalidGethNode
	}

	if accountManager == nil {
		return "", "", ErrInvalidAccountManager
	}

	// re-create extended key (see BIP32)
	m := extkeys.NewMnemonic()
	extKey, err := extkeys.NewMaster(m.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
	if err != nil {
		return "", "", errextra.Wrap(err, "Can not create master extended key")
	}

	// import re-created key into account keystore
	address, pubKey, err = importExtendedKey(extKey, password)
	if err != nil {
		return
	}

	return address, pubKey, nil
}

// selectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
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

	// persist address for easier recovery of currently selected key (from Whisper)
	selectedAddress = address

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

	selectedAddress = ""

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

// importExtendedKey processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
func importExtendedKey(extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
	// imports extended key, create key file (if necessary)
	account, err := accountManager.ImportExtendedKey(extKey, password)
	if err != nil {
		return "", "", errextra.Wrap(err, "Account manager could not create the account")
	}
	address = fmt.Sprintf("%x", account.Address)

	// obtain public key to return
	account, key, err := accountManager.AccountDecryptedKey(account, password)
	if err != nil {
		return address, "", errextra.Wrap(err, "Could not recover the key")
	}
	pubKey = common.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
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
