package geth

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
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

// CreateAccount creates an internal geth account
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

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	nodeManager := GetNodeManager()
	accountManager, err := nodeManager.AccountManager()
	if err != nil {
		return "", "", err
	}

	if parentAddress == "" && nodeManager.SelectedAccount != nil { // derive from selected account by default
		parentAddress = string(nodeManager.SelectedAccount.Address.Hex())
	}

	if parentAddress == "" {
		return "", "", ErrNoAccountSelected
	}

	account, err := utils.MakeAddress(accountManager, parentAddress)
	if err != nil {
		return "", "", ErrAddressToAccountMappingFailure
	}

	// make sure that given password can decrypt key associated with a given parent address
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
	accountKey.SubAccountIndex++

	// import derived key into account keystore
	address, pubKey, err = importExtendedKey(childKey, password)
	if err != nil {
		return
	}

	// update in-memory selected account
	if nodeManager.SelectedAccount != nil {
		nodeManager.SelectedAccount.AccountKey = accountKey
	}

	return address, pubKey, nil
}

// RecoverAccount re-creates master key using given details.
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

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
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

	// persist account key for easier recovery of currently selected key
	subAccounts, err := findSubAccounts(accountKey.ExtendedKey, accountKey.SubAccountIndex)
	if err != nil {
		return err
	}
	nodeManager.SelectedAccount = &SelectedExtKey{
		Address:     account.Address,
		AccountKey:  accountKey,
		SubAccounts: subAccounts,
	}

	return nil
}

// Logout clears whisper identities
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

	nodeManager.SelectedAccount = nil

	return nil
}

// UnlockAccount unlocks an existing account for a certain duration and
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

func onAccountsListRequest(entities []accounts.Account) []accounts.Account {
	nodeManager := GetNodeManager()

	if nodeManager.SelectedAccount == nil {
		return []accounts.Account{}
	}

	refreshSelectedAccount()

	filtered := make([]accounts.Account, 0)
	for _, account := range entities {
		// main account
		if nodeManager.SelectedAccount.Address.Hex() == account.Address.Hex() {
			filtered = append(filtered, account)
		} else {
			// sub accounts
			for _, subAccount := range nodeManager.SelectedAccount.SubAccounts {
				if subAccount.Address.Hex() == account.Address.Hex() {
					filtered = append(filtered, account)
				}
			}
		}
	}

	return filtered
}

// refreshSelectedAccount re-populates list of sub-accounts of the currently selected account (if any)
func refreshSelectedAccount() {
	nodeManager := GetNodeManager()

	if nodeManager.SelectedAccount == nil {
		return
	}

	accountKey := nodeManager.SelectedAccount.AccountKey
	if accountKey == nil {
		return
	}

	// re-populate list of sub-accounts
	subAccounts, err := findSubAccounts(accountKey.ExtendedKey, accountKey.SubAccountIndex)
	if err != nil {
		return
	}
	nodeManager.SelectedAccount = &SelectedExtKey{
		Address:     nodeManager.SelectedAccount.Address,
		AccountKey:  nodeManager.SelectedAccount.AccountKey,
		SubAccounts: subAccounts,
	}
}

// findSubAccounts traverses cached accounts and adds as a sub-accounts any
// that belong to the currently selected account.
// The extKey is CKD#2 := root of sub-accounts of the main account
func findSubAccounts(extKey *extkeys.ExtendedKey, subAccountIndex uint32) ([]accounts.Account, error) {
	nodeManager := GetNodeManager()
	accountManager, err := nodeManager.AccountManager()
	if err != nil {
		return []accounts.Account{}, err
	}

	subAccounts := make([]accounts.Account, 0)
	if extKey.Depth == 5 { // CKD#2 level
		// gather possible sub-account addresses
		subAccountAddresses := make([]common.Address, 0)
		for i := uint32(0); i < subAccountIndex; i++ {
			childKey, err := extKey.Child(i)
			if err != nil {
				return []accounts.Account{}, err
			}
			subAccountAddresses = append(subAccountAddresses, crypto.PubkeyToAddress(childKey.ToECDSA().PublicKey))
		}

		// see if any of the gathered addresses actually exist in cached accounts list
		for _, cachedAccount := range accountManager.Accounts() {
			for _, possibleAddress := range subAccountAddresses {
				if possibleAddress.Hex() == cachedAccount.Address.Hex() {
					subAccounts = append(subAccounts, cachedAccount)
				}
			}
		}
	}

	return subAccounts, nil
}
