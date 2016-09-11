package geth

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/extkeys"
)

var (
	ErrAddressToAccountMappingFailure  = errors.New("cannot retreive a valid account for a given address")
	ErrAccountToKeyMappingFailure      = errors.New("cannot retreive a valid key for a given account")
	ErrUnlockCalled                    = errors.New("no need to unlock accounts, login instead")
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	ErrWhisperClearIdentitiesFailure   = errors.New("failed to clear whisper identities")
	ErrWhisperNoIdentityFound          = errors.New("failed to locate identity previously injected into Whisper")
	ErrNoAccountSelected               = errors.New("no account has been selected, please login")
	ErrInvalidMasterKeyCreated         = errors.New("can not create master extended key")
)

// createAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func CreateAccount(password string) (address, pubKey, mnemonic string, err error) {
	// generate mnemonic phrase
	m := extkeys.NewMnemonic(extkeys.Salt)
	mnemonic, err = m.MnemonicPhrase(128, extkeys.EnglishLanguage)
	if err != nil {
		return "", "", "", fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	// generate extended master key (see BIP32)
	extKey, err := extkeys.NewMaster(m.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
	if err != nil {
		return "", "", "", fmt.Errorf("can not create master extended key: %v", err)
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
func CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	nodeManager := GetNodeManager()
	accountManager, err := nodeManager.AccountManager()
	if err != nil {
		return "", "", err
	}

	if parentAddress == "" { // by default derive from currently selected account
		parentAddress = nodeManager.SelectedAddress
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
func RecoverAccount(password, mnemonic string) (address, pubKey string, err error) {
	// re-create extended key (see BIP32)
	m := extkeys.NewMnemonic(extkeys.Salt)
	extKey, err := extkeys.NewMaster(m.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
	if err != nil {
		return "", "", ErrInvalidMasterKeyCreated
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
func SelectAccount(address, password string) error {
	nodeManager := GetNodeManager()
	accountManager, err := nodeManager.AccountManager()
	if err != nil {
		return err
	}

	account, err := utils.MakeAddress(accountManager, address)
	if err != nil {
		return ErrAddressToAccountMappingFailure
	}

	account, accountKey, err := accountManager.AccountDecryptedKey(account, password)
	if err != nil {
		return fmt.Errorf("%s: %v", ErrAccountToKeyMappingFailure.Error(), err)
	}

	whisperService, err := nodeManager.WhisperService()
	if err != nil {
		return err
	}

	if err := whisperService.InjectIdentity(accountKey.PrivateKey); err != nil {
		return ErrWhisperIdentityInjectionFailure
	}

	// persist address for easier recovery of currently selected key (from Whisper)
	nodeManager.SelectedAddress = address

	return nil
}

// logout clears whisper identities
func Logout() error {
	nodeManager := GetNodeManager()
	whisperService, err := nodeManager.WhisperService()
	if err != nil {
		return err
	}

	err = whisperService.ClearIdentities()
	if err != nil {
		return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
	}

	nodeManager.SelectedAddress = ""

	return nil
}

// unlockAccount unlocks an existing account for a certain duration and
// inject the account as a whisper identity if the account was created as
// a whisper enabled account
func UnlockAccount(address, password string, seconds int) error {
	return ErrUnlockCalled
}

// importExtendedKey processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
func importExtendedKey(extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
	accountManager, err := GetNodeManager().AccountManager()
	if err != nil {
		return "", "", err
	}

	// imports extended key, create key file (if necessary)
	account, err := accountManager.ImportExtendedKey(extKey, password)
	if err != nil {
		return "", "", err
	}
	address = fmt.Sprintf("%x", account.Address)

	// obtain public key to return
	account, key, err := accountManager.AccountDecryptedKey(account, password)
	if err != nil {
		return address, "", err
	}
	pubKey = common.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}
