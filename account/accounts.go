package account

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/extkeys"
	protocol "github.com/status-im/status-go/protocol/types"
)

// errors
var (
	ErrAddressToAccountMappingFailure = errors.New("cannot retrieve a valid account for a given address")
	ErrAccountToKeyMappingFailure     = errors.New("cannot retrieve a valid key for a given account")
	ErrNoAccountSelected              = errors.New("no account has been selected, please login")
	ErrInvalidMasterKeyCreated        = errors.New("can not create master extended key")
	ErrOnboardingNotStarted           = errors.New("onboarding must be started before choosing an account")
	ErrOnboardingAccountNotFound      = errors.New("cannot find onboarding account with the given id")
	ErrAccountKeyStoreMissing         = errors.New("account key store is not set")
)

var zeroAddress = common.Address{}

// Manager represents account manager interface.
type Manager struct {
	mu       sync.RWMutex
	keystore *keystore.KeyStore
	manager  *accounts.Manager

	accountsGenerator *generator.Generator
	onboarding        *Onboarding

	selectedChatAccount *SelectedExtKey // account that was processed during the last call to SelectAccount()
	mainAccountAddress  common.Address
	watchAddresses      []common.Address
}

// NewManager returns new node account manager.
func NewManager() *Manager {
	m := &Manager{}
	m.accountsGenerator = generator.New(m)
	return m
}

// InitKeystore sets key manager and key store.
func (m *Manager) InitKeystore(keydir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	manager, err := makeAccountManager(keydir)
	if err != nil {
		return err
	}
	m.manager = manager
	backends := manager.Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		return ErrAccountKeyStoreMissing
	}
	keyStore, ok := backends[0].(*keystore.KeyStore)
	if !ok {
		return ErrAccountKeyStoreMissing
	}
	m.keystore = keyStore
	return nil
}

func (m *Manager) GetKeystore() *keystore.KeyStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keystore
}

func (m *Manager) GetManager() *accounts.Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.manager
}

// AccountsGenerator returns accountsGenerator.
func (m *Manager) AccountsGenerator() *generator.Generator {
	return m.accountsGenerator
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (m *Manager) CreateAccount(password string) (Info, string, error) {
	info := Info{}
	// generate mnemonic phrase
	mn := extkeys.NewMnemonic()
	mnemonic, err := mn.MnemonicPhrase(extkeys.EntropyStrength128, extkeys.EnglishLanguage)
	if err != nil {
		return info, "", fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	// Generate extended master key (see BIP32)
	// We call extkeys.NewMaster with a seed generated with the 12 mnemonic words
	// but without using the optional password as an extra entropy as described in BIP39.
	// Future ideas/iterations in Status can add an an advanced options
	// for expert users, to be able to add a passphrase to the generation of the seed.
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, ""))
	if err != nil {
		return info, "", fmt.Errorf("can not create master extended key: %v", err)
	}

	// import created key into account keystore
	info.WalletAddress, info.WalletPubKey, err = m.importExtendedKey(extkeys.KeyPurposeWallet, extKey, password)
	if err != nil {
		return info, "", err
	}

	info.ChatAddress = info.WalletAddress
	info.ChatPubKey = info.WalletPubKey

	return info, mnemonic, nil
}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (m *Manager) RecoverAccount(password, mnemonic string) (Info, error) {
	info := Info{}
	// re-create extended key (see BIP32)
	mn := extkeys.NewMnemonic()
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, ""))
	if err != nil {
		return info, ErrInvalidMasterKeyCreated
	}

	// import re-created key into account keystore
	info.WalletAddress, info.WalletPubKey, err = m.importExtendedKey(extkeys.KeyPurposeWallet, extKey, password)
	if err != nil {
		return info, err
	}

	info.ChatAddress = info.WalletAddress
	info.ChatPubKey = info.WalletPubKey

	return info, nil
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
func (m *Manager) SelectAccount(loginParams LoginParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.accountsGenerator.Reset()

	selectedChatAccount, err := m.unlockExtendedKey(loginParams.ChatAddress.String(), loginParams.Password)
	if err != nil {
		return err
	}
	m.watchAddresses = loginParams.WatchAddresses
	m.mainAccountAddress = loginParams.MainAccount
	m.selectedChatAccount = selectedChatAccount
	return nil
}

func (m *Manager) SetAccountAddresses(main common.Address, secondary ...common.Address) {
	m.watchAddresses = []common.Address{main}
	m.watchAddresses = append(m.watchAddresses, secondary...)
	m.mainAccountAddress = main
}

// SetChatAccount initializes selectedChatAccount with privKey
func (m *Manager) SetChatAccount(privKey *ecdsa.PrivateKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	address := crypto.PubkeyToAddress(privKey.PublicKey)
	id := uuid.NewRandom()
	key := &keystore.Key{
		Id:         id,
		Address:    address,
		PrivateKey: privKey,
	}

	m.selectedChatAccount = &SelectedExtKey{
		Address:    address,
		AccountKey: key,
	}
}

// MainAccountAddress returns currently selected watch addresses.
func (m *Manager) MainAccountAddress() (common.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.mainAccountAddress == zeroAddress {
		return zeroAddress, ErrNoAccountSelected
	}

	return m.mainAccountAddress, nil
}

// WatchAddresses returns currently selected watch addresses.
func (m *Manager) WatchAddresses() []common.Address {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.watchAddresses
}

// SelectedChatAccount returns currently selected chat account
func (m *Manager) SelectedChatAccount() (*SelectedExtKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.selectedChatAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedChatAccount, nil
}

// Logout clears selected accounts.
func (m *Manager) Logout() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.accountsGenerator.Reset()
	m.mainAccountAddress = zeroAddress
	m.watchAddresses = nil
	m.selectedChatAccount = nil
}

// ImportAccount imports the account specified with privateKey.
func (m *Manager) ImportAccount(privateKey *ecdsa.PrivateKey, password string) (common.Address, error) {
	if m.keystore == nil {
		return common.Address{}, ErrAccountKeyStoreMissing
	}

	account, err := m.keystore.ImportECDSA(privateKey, password)

	return account.Address, err
}

func (m *Manager) ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
	if m.keystore == nil {
		return "", "", ErrAccountKeyStoreMissing
	}

	// imports extended key, create key file (if necessary)
	account, err := m.keystore.ImportSingleExtendedKey(extKey, password)
	if err != nil {
		return "", "", err
	}

	address = account.Address.Hex()

	// obtain public key to return
	account, key, err := m.keystore.AccountDecryptedKey(account, password)
	if err != nil {
		return address, "", err
	}

	pubKey = protocol.EncodeHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}

// importExtendedKey processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
func (m *Manager) importExtendedKey(keyPurpose extkeys.KeyPurpose, extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
	if m.keystore == nil {
		return "", "", ErrAccountKeyStoreMissing
	}

	// imports extended key, create key file (if necessary)
	account, err := m.keystore.ImportExtendedKeyForPurpose(keyPurpose, extKey, password)
	if err != nil {
		return "", "", err
	}
	address = account.Address.Hex()

	// obtain public key to return
	account, key, err := m.keystore.AccountDecryptedKey(account, password)
	if err != nil {
		return address, "", err
	}
	pubKey = protocol.EncodeHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}

// Accounts returns list of addresses for selected account, including
// subaccounts.
func (m *Manager) Accounts() ([]gethcommon.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	addresses := make([]gethcommon.Address, 0)
	if m.mainAccountAddress != zeroAddress {
		addresses = append(addresses, m.mainAccountAddress)
	}

	return addresses, nil
}

// StartOnboarding starts the onboarding process generating accountsCount accounts and returns a slice of OnboardingAccount.
func (m *Manager) StartOnboarding(accountsCount, mnemonicPhraseLength int) ([]*OnboardingAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	onboarding, err := NewOnboarding(accountsCount, mnemonicPhraseLength)
	if err != nil {
		return nil, err
	}

	m.onboarding = onboarding

	return m.onboarding.Accounts(), nil
}

// RemoveOnboarding reset the current onboarding struct setting it to nil and deleting the accounts from memory.
func (m *Manager) RemoveOnboarding() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onboarding = nil
}

// ImportOnboardingAccount imports the account specified by id and encrypts it with password.
func (m *Manager) ImportOnboardingAccount(id string, password string) (Info, string, error) {
	var info Info

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.onboarding == nil {
		return info, "", ErrOnboardingNotStarted
	}

	acc, err := m.onboarding.Account(id)
	if err != nil {
		return info, "", err
	}

	info, err = m.RecoverAccount(password, acc.mnemonic)
	if err != nil {
		return info, "", err
	}

	m.onboarding = nil

	return info, acc.mnemonic, nil
}

// AddressToDecryptedAccount tries to load decrypted key for a given account.
// The running node, has a keystore directory which is loaded on start. Key file
// for a given address is expected to be in that directory prior to node start.
func (m *Manager) AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error) {
	if m.keystore == nil {
		return accounts.Account{}, nil, ErrAccountKeyStoreMissing
	}

	account, err := ParseAccountString(address)
	if err != nil {
		return accounts.Account{}, nil, ErrAddressToAccountMappingFailure
	}

	var key *keystore.Key
	account, key, err = m.keystore.AccountDecryptedKey(account, password)
	if err != nil {
		err = fmt.Errorf("%s: %s", ErrAccountToKeyMappingFailure, err)
	}

	return account, key, err
}

func (m *Manager) unlockExtendedKey(address, password string) (*SelectedExtKey, error) {
	account, accountKey, err := m.AddressToDecryptedAccount(address, password)
	if err != nil {
		return nil, err
	}

	selectedExtendedKey := &SelectedExtKey{
		Address:    account.Address,
		AccountKey: accountKey,
	}

	return selectedExtendedKey, nil
}
