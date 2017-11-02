package account

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/extkeys"
	"github.com/status-im/status-go/geth/common"
)

// errors
var (
	ErrAddressToAccountMappingFailure  = errors.New("cannot retrieve a valid account for a given address")
	ErrAccountToKeyMappingFailure      = errors.New("cannot retrieve a valid key for a given account")
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	ErrWhisperClearIdentitiesFailure   = errors.New("failed to clear whisper identities")
	ErrNoAccountSelected               = errors.New("no account has been selected, please login")
	ErrInvalidMasterKeyCreated         = errors.New("can not create master extended key")
)

// Manager represents account manager interface
type Manager struct {
	node            accountNode
	selectedAccount *common.SelectedExtKey // account that was processed during the last call to SelectAccount()

	extKeyImporter   extendedKeyImporter
	subAccountFinder subAccountFinder
}

// NewManager returns new node account manager
func NewManager(nodeManager common.NodeManager) *Manager {
	node := newAccountNodeManager(nodeManager)
	return &Manager{
		node:             node,
		extKeyImporter:   new(extendedKeyImporterBase),
		subAccountFinder: new(subAccountFinderBase),
	}
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (m *Manager) CreateAccount(password string) (address, pubKey, mnemonic string, err error) {
	// generate mnemonic phrase
	mn := extkeys.NewMnemonic(extkeys.Salt)
	mnemonic, err = mn.MnemonicPhrase(128, extkeys.EnglishLanguage)
	if err != nil {
		return "", "", "", fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	// generate extended master key (see BIP32)
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
	if err != nil {
		return "", "", "", fmt.Errorf("can not create master extended key: %v", err)
	}

	keyStore, err := m.node.AccountKeyStore()
	if err != nil {
		return "", "", "", err
	}

	// import created key into account keystore
	address, pubKey, err = m.extKeyImporter.Import(keyStore, extKey, password)
	if err != nil {
		return "", "", "", err
	}

	return address, pubKey, mnemonic, nil
}

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func (m *Manager) CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	keyStore, err := m.node.AccountKeyStore()
	if err != nil {
		return "", "", err
	}

	if parentAddress == "" && m.selectedAccount != nil { // derive from selected account by default
		parentAddress = m.selectedAccount.Address.Hex()
	}

	if parentAddress == "" {
		return "", "", ErrNoAccountSelected
	}

	account, err := common.ParseAccountString(parentAddress)
	if err != nil {
		return "", "", ErrAddressToAccountMappingFailure
	}

	// make sure that given password can decrypt key associated with a given parent address
	account, accountKey, err := keyStore.AccountDecryptedKey(account, password)
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

	if err = keyStore.IncSubAccountIndex(account, password); err != nil {
		return "", "", err
	}
	accountKey.SubAccountIndex++

	// import derived key into account keystore
	address, pubKey, err = m.extKeyImporter.Import(keyStore, childKey, password)
	if err != nil {
		return
	}

	// update in-memory selected account
	if m.selectedAccount != nil {
		m.selectedAccount.AccountKey = accountKey
	}

	return address, pubKey, nil
}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (m *Manager) RecoverAccount(password, mnemonic string) (address, pubKey string, err error) {
	// re-create extended key (see BIP32)
	mn := extkeys.NewMnemonic(extkeys.Salt)
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, password), []byte(extkeys.Salt))
	if err != nil {
		return "", "", ErrInvalidMasterKeyCreated
	}

	keyStore, err := m.node.AccountKeyStore()
	if err != nil {
		return "", "", err
	}

	// import re-created key into account keystore
	return m.extKeyImporter.Import(keyStore, extKey, password)
}

// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
// If no error is returned, then account is considered verified.
func (m *Manager) VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error) {
	addressObj := gethcommon.BytesToAddress(gethcommon.FromHex(address))

	foundKeyFile, err := keyfileFinder.Find(keyStoreDir, addressObj)
	if err != nil {
		return nil, fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	if len(foundKeyFile) == 0 {
		return nil, fmt.Errorf("cannot locate account for address: %s", addressObj.Hex())
	}

	key, err := keystore.DecryptKey(foundKeyFile, password)
	if err != nil {
		return nil, err
	}

	// avoid swap attack
	if key.Address != addressObj {
		return nil, fmt.Errorf("account mismatch: have %s, want %s", key.Address.Hex(), addressObj.Hex())
	}

	return key, nil
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (m *Manager) SelectAccount(address, password string) error {
	keyStore, err := m.node.AccountKeyStore()
	if err != nil {
		return err
	}

	account, err := common.ParseAccountString(address)
	if err != nil {
		return ErrAddressToAccountMappingFailure
	}

	account, accountKey, err := keyStore.AccountDecryptedKey(account, password)
	if err != nil {
		return fmt.Errorf("%s: %v", ErrAccountToKeyMappingFailure.Error(), err)
	}

	whisperService, err := m.node.WhisperService()
	if err != nil {
		return err
	}

	err = whisperService.SelectKeyPair(accountKey.PrivateKey)
	if err != nil {
		return ErrWhisperIdentityInjectionFailure
	}

	// persist account key for easier recovery of currently selected key
	subAccounts, err := m.subAccountFinder.Find(keyStore, accountKey.ExtendedKey, accountKey.SubAccountIndex)
	if err != nil {
		return err
	}
	m.selectedAccount = &common.SelectedExtKey{
		Address:     account.Address,
		AccountKey:  accountKey,
		SubAccounts: subAccounts,
	}

	return nil
}

// SelectedAccount returns currently selected account
func (m *Manager) SelectedAccount() (*common.SelectedExtKey, error) {
	if m.selectedAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedAccount, nil
}

// ReSelectAccount selects previously selected account, often, after node restart.
func (m *Manager) ReSelectAccount() error {
	selectedAccount := m.selectedAccount
	if selectedAccount == nil {
		return nil
	}

	whisperService, err := m.node.WhisperService()
	if err != nil {
		return err
	}

	if err := whisperService.SelectKeyPair(selectedAccount.AccountKey.PrivateKey); err != nil {
		return ErrWhisperIdentityInjectionFailure
	}

	return nil
}

// Logout clears whisper identities
func (m *Manager) Logout() error {
	whisperService, err := m.node.WhisperService()
	if err != nil {
		return err
	}

	err = whisperService.DeleteKeyPairs()
	if err != nil {
		return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
	}

	m.selectedAccount = nil

	return nil
}

// Accounts returns list of addresses for selected account, including
// subaccounts.
func (m *Manager) Accounts() ([]gethcommon.Address, error) {
	am, err := m.node.AccountManager()
	if err != nil {
		return nil, err
	}

	var addresses []gethcommon.Address
	for _, wallet := range am.Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}

	if m.selectedAccount == nil {
		return []gethcommon.Address{}, nil
	}

	//update info about subaccounts into selected account
	m.refreshSelectedAccount()

	filtered := make([]gethcommon.Address, 0)
	for _, account := range addresses {
		// main account
		if m.selectedAccount.Address.Hex() == account.Hex() {
			filtered = append(filtered, account)
		} else {
			// sub accounts
			for _, subAccount := range m.selectedAccount.SubAccounts {
				if subAccount.Address.Hex() == account.Hex() {
					filtered = append(filtered, account)
				}
			}
		}
	}

	return filtered, nil
}

// refreshSelectedAccount re-populates list of sub-accounts of the currently selected account (if any)
func (m *Manager) refreshSelectedAccount() {
	if m.selectedAccount == nil {
		return
	}

	accountKey := m.selectedAccount.AccountKey
	if accountKey == nil {
		return
	}

	keyStore, err := m.node.AccountKeyStore()
	if err != nil {
		return
	}
	// re-populate list of sub-accounts
	subAccounts, err := m.subAccountFinder.Find(keyStore, accountKey.ExtendedKey, accountKey.SubAccountIndex)
	if err != nil {
		return
	}
	m.selectedAccount = &common.SelectedExtKey{
		Address:     m.selectedAccount.Address,
		AccountKey:  m.selectedAccount.AccountKey,
		SubAccounts: subAccounts,
	}
}

// AddressToDecryptedAccount tries to load decrypted key for a given account.
// The running node, has a keystore directory which is loaded on start. Key file
// for a given address is expected to be in that directory prior to node start.
func (m *Manager) AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error) {
	keyStore, err := m.node.AccountKeyStore()
	if err != nil {
		return accounts.Account{}, nil, err
	}

	account, err := common.ParseAccountString(address)
	if err != nil {
		return accounts.Account{}, nil, ErrAddressToAccountMappingFailure
	}

	return keyStore.AccountDecryptedKey(account, password)
}
