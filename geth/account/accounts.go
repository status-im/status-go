package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/extkeys"
)

// errors
var (
	ErrAddressToAccountMappingFailure = errors.New("cannot retrieve a valid account for a given address")
	ErrAccountToKeyMappingFailure     = errors.New("cannot retrieve a valid key for a given account")
	ErrNoAccountSelected              = errors.New("no account has been selected, please login")
	ErrInvalidMasterKeyCreated        = errors.New("can not create master extended key")
)

// GethServiceProvider provides required geth services.
type GethServiceProvider interface {
	AccountManager() (*accounts.Manager, error)
	AccountKeyStore() (*keystore.KeyStore, error)
}

// Manager represents account manager interface.
type Manager struct {
	geth            GethServiceProvider
	selectedAccount *SelectedExtKey // account that was processed during the last call to SelectAccount()
}

// NewManager returns new node account manager.
func NewManager(geth GethServiceProvider) *Manager {
	return &Manager{
		geth: geth,
	}
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (m *Manager) CreateAccount(password string) (address, pubKey, mnemonic string, err error) {
	// generate mnemonic phrase
	mn := extkeys.NewMnemonic()
	mnemonic, err = mn.MnemonicPhrase(extkeys.EntropyStrength128, extkeys.EnglishLanguage)
	if err != nil {
		return "", "", "", fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	// generate extended master key (see BIP32)
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, ""))
	if err != nil {
		return "", "", "", fmt.Errorf("can not create master extended key: %v", err)
	}

	// import created key into account keystore
	address, pubKey, err = m.importExtendedKey(extKey, password)
	if err != nil {
		return "", "", "", err
	}

	return address, pubKey, mnemonic, nil
}

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func (m *Manager) CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	keyStore, err := m.geth.AccountKeyStore()
	if err != nil {
		return "", "", err
	}

	if parentAddress == "" && m.selectedAccount != nil { // derive from selected account by default
		parentAddress = m.selectedAccount.Address.Hex()
	}

	if parentAddress == "" {
		return "", "", ErrNoAccountSelected
	}

	account, err := ParseAccountString(parentAddress)
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
	address, pubKey, err = m.importExtendedKey(childKey, password)
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
	mn := extkeys.NewMnemonic()
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, ""))
	if err != nil {
		return "", "", ErrInvalidMasterKeyCreated
	}

	// import re-created key into account keystore
	address, pubKey, err = m.importExtendedKey(extKey, password)
	if err != nil {
		return
	}

	return address, pubKey, nil
}

// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
// If no error is returned, then account is considered verified.
func (m *Manager) VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error) {
	var err error
	var foundKeyFile []byte

	addressObj := gethcommon.BytesToAddress(gethcommon.FromHex(address))
	checkAccountKey := func(path string, fileInfo os.FileInfo) error {
		if len(foundKeyFile) > 0 || fileInfo.IsDir() {
			return nil
		}

		rawKeyFile, e := ioutil.ReadFile(path)
		if e != nil {
			return fmt.Errorf("invalid account key file: %v", e)
		}

		var accountKey struct {
			Address string `json:"address"`
		}
		if e := json.Unmarshal(rawKeyFile, &accountKey); e != nil {
			return fmt.Errorf("failed to read key file: %s", e)
		}

		if gethcommon.HexToAddress("0x"+accountKey.Address).Hex() == addressObj.Hex() {
			foundKeyFile = rawKeyFile
		}

		return nil
	}
	// locate key within key store directory (address should be within the file)
	err = filepath.Walk(keyStoreDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return checkAccountKey(path, fileInfo)
	})
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
// using provided password. Once verification is done, all previous identities are removed).
func (m *Manager) SelectAccount(address, password string) error {
	keyStore, err := m.geth.AccountKeyStore()
	if err != nil {
		return err
	}

	account, err := ParseAccountString(address)
	if err != nil {
		return ErrAddressToAccountMappingFailure
	}

	account, accountKey, err := keyStore.AccountDecryptedKey(account, password)
	if err != nil {
		return fmt.Errorf("%s: %v", ErrAccountToKeyMappingFailure.Error(), err)
	}

	// persist account key for easier recovery of currently selected key
	subAccounts, err := m.findSubAccounts(accountKey.ExtendedKey, accountKey.SubAccountIndex)
	if err != nil {
		return err
	}
	m.selectedAccount = &SelectedExtKey{
		Address:     account.Address,
		AccountKey:  accountKey,
		SubAccounts: subAccounts,
	}

	return nil
}

// SelectedAccount returns currently selected account
func (m *Manager) SelectedAccount() (*SelectedExtKey, error) {
	if m.selectedAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedAccount, nil
}

// Logout clears selectedAccount.
func (m *Manager) Logout() error {
	m.selectedAccount = nil

	return nil
}

// importExtendedKey processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
func (m *Manager) importExtendedKey(extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
	keyStore, err := m.geth.AccountKeyStore()
	if err != nil {
		return "", "", err
	}

	// imports extended key, create key file (if necessary)
	account, err := keyStore.ImportExtendedKey(extKey, password)
	if err != nil {
		return "", "", err
	}
	address = account.Address.Hex()

	// obtain public key to return
	account, key, err := keyStore.AccountDecryptedKey(account, password)
	if err != nil {
		return address, "", err
	}
	pubKey = gethcommon.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}

// Accounts returns list of addresses for selected account, including
// subaccounts.
func (m *Manager) Accounts() ([]gethcommon.Address, error) {
	am, err := m.geth.AccountManager()
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

	// re-populate list of sub-accounts
	subAccounts, err := m.findSubAccounts(accountKey.ExtendedKey, accountKey.SubAccountIndex)
	if err != nil {
		return
	}
	m.selectedAccount = &SelectedExtKey{
		Address:     m.selectedAccount.Address,
		AccountKey:  m.selectedAccount.AccountKey,
		SubAccounts: subAccounts,
	}
}

// findSubAccounts traverses cached accounts and adds as a sub-accounts any
// that belong to the currently selected account.
// The extKey is CKD#2 := root of sub-accounts of the main account
func (m *Manager) findSubAccounts(extKey *extkeys.ExtendedKey, subAccountIndex uint32) ([]accounts.Account, error) {
	keyStore, err := m.geth.AccountKeyStore()
	if err != nil {
		return []accounts.Account{}, err
	}

	subAccounts := make([]accounts.Account, 0)
	if extKey.Depth == 5 { // CKD#2 level
		// gather possible sub-account addresses
		subAccountAddresses := make([]gethcommon.Address, 0)
		for i := uint32(0); i < subAccountIndex; i++ {
			childKey, err := extKey.Child(i)
			if err != nil {
				return []accounts.Account{}, err
			}
			subAccountAddresses = append(subAccountAddresses, crypto.PubkeyToAddress(childKey.ToECDSA().PublicKey))
		}

		// see if any of the gathered addresses actually exist in cached accounts list
		for _, cachedAccount := range keyStore.Accounts() {
			for _, possibleAddress := range subAccountAddresses {
				if possibleAddress.Hex() == cachedAccount.Address.Hex() {
					subAccounts = append(subAccounts, cachedAccount)
				}
			}
		}
	}

	return subAccounts, nil
}

// AddressToDecryptedAccount tries to load decrypted key for a given account.
// The running node, has a keystore directory which is loaded on start. Key file
// for a given address is expected to be in that directory prior to node start.
func (m *Manager) AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error) {
	keyStore, err := m.geth.AccountKeyStore()
	if err != nil {
		return accounts.Account{}, nil, err
	}

	account, err := ParseAccountString(address)
	if err != nil {
		return accounts.Account{}, nil, ErrAddressToAccountMappingFailure
	}

	return keyStore.AccountDecryptedKey(account, password)
}
