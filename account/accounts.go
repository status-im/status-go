package account

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/status-im/status-go/account/common"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/account/keystore/geth"
	"github.com/status-im/status-go/account/types"
	gocommon "github.com/status-im/status-go/common"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

type ErrCannotLocateKeyFile struct {
	Msg string
}

func (e ErrCannotLocateKeyFile) Error() string {
	return e.Msg
}

// DefaultManager represents default account manager implementation
type DefaultManager struct {
	keystoreMu sync.RWMutex
	keystore   types.KeyStore

	selectedChatAccountMutex sync.RWMutex
	selectedChatAccount      *generator.Account // account that was processed during the last call to SelectAccount()

	logger *zap.Logger
}

func NewDefaultManager(logger *zap.Logger) *DefaultManager {
	return &DefaultManager{
		logger: logger,
	}
}

func (m *DefaultManager) IsKeystoreSet() bool {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	return m.keystore != nil
}

func (m *DefaultManager) SetKeystore(keystore types.KeyStore) {
	m.keystoreMu.Lock()
	defer m.keystoreMu.Unlock()

	m.keystore = keystore
}

// CreateAndStoreAccount creates an internal geth account and stores it in the keystore
func (m *DefaultManager) CreateAndStoreAccount(password string) (genAccount *generator.Account, mnemonic string, err error) {
	mnemonic, err = common.CreateRandomMnemonicWithDefaultLength()
	if err != nil {
		return
	}

	genAccount, err = m.CreateFromMnemonicAndStoreAccount(mnemonic, password)

	return
}

// CreateFromMnemonicAndStoreAccount creates an internal geth account from a mnemonic and stores it in the keystore
func (m *DefaultManager) CreateFromMnemonicAndStoreAccount(mnemonic string, password string) (genAccount *generator.Account, err error) {
	genAccount, err = generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return
	}

	err = m.storeToKeystore(genAccount, password)

	return
}

func (m *DefaultManager) CreateFromPrivateKeyAndStoreAccount(privateKey string, password string) (genAccount *generator.Account, err error) {
	acc, err := generator.CreateAccountFromPrivateKey(privateKey)
	if err != nil {
		return
	}

	err = m.storeToKeystore(acc, password)

	return
}

// VerifyAccountPassword verifies if the account key for a given address and password is correct.
func (m *DefaultManager) VerifyAccountPassword(address ethtypes.Address, password string) (bool, error) {
	account, err := m.LoadAccount(address, password)
	if err != nil {
		return false, err
	}

	if account.Address() != address {
		return false, fmt.Errorf("account mismatch: have %s, want %s", gocommon.TruncateWithDot(account.Address().Hex()), gocommon.TruncateWithDot(address.Hex()))
	}

	return true, nil
}

// LoadAccount loads an account key from the keystore for a given address and password.
// If either address or password is incorrect, an error is returned.
func (m *DefaultManager) LoadAccount(address ethtypes.Address, password string) (*generator.Account, error) {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return nil, ErrAccountKeyStoreMissing
	}

	_, privateKey, extendedKey, err := m.keystore.AccountDecryptedKey(address, password)
	if err != nil {
		if errors.Is(err, geth.ErrNoMatch) {
			return nil, &ErrCannotLocateKeyFile{fmt.Sprintf("cannot locate account for address: %s", address.Hex())}
		}
		return nil, err
	}

	account := generator.NewAccount(privateKey, extendedKey)
	if err := account.ValidateExtendedKey(); err != nil {
		return nil, err
	}

	return account, nil
}

func (m *DefaultManager) storeToKeystore(acc *generator.Account, password string) (err error) {
	m.keystoreMu.Lock()
	defer m.keystoreMu.Unlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}

	if acc.HasExtendedKey() {
		_, err = m.keystore.ImportSingleExtendedKey(acc.ExtendedKey(), password)
		return
	}

	_, err = m.keystore.ImportECDSA(acc.PrivateKey(), password)
	return
}

func (m *DefaultManager) setChatAccount(account *generator.Account) {
	m.selectedChatAccountMutex.Lock()
	defer m.selectedChatAccountMutex.Unlock()

	m.selectedChatAccount = account
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, all previous identities are removed).
func (m *DefaultManager) SelectAccount(loginParams types.LoginParams) error {
	selectedChatAccount, err := m.LoadAccount(loginParams.ChatAddress, loginParams.Password)
	if err != nil {
		return err
	}

	m.setChatAccount(selectedChatAccount)

	return nil
}

// SetChatAccount initializes selectedChatAccount with privKey
func (m *DefaultManager) SetChatAccount(privKey *ecdsa.PrivateKey) error {
	account := generator.NewAccount(privKey, nil)

	m.setChatAccount(account)

	return nil
}

// SelectedChatAccount returns currently selected chat account
func (m *DefaultManager) SelectedChatAccount() (*generator.Account, error) {
	m.selectedChatAccountMutex.RLock()
	defer m.selectedChatAccountMutex.RUnlock()

	if m.selectedChatAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedChatAccount, nil
}

// Logout clears everything.
func (m *DefaultManager) Logout() {
	m.setChatAccount(nil)
	m.SetKeystore(nil)
}

// Accounts returns list of addresses for selected account, including subaccounts.
func (m *DefaultManager) Accounts() ([]ethtypes.Address, error) {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return nil, ErrAccountKeyStoreMissing
	}

	ksAccounts := m.keystore.Accounts()
	addresses := make([]ethtypes.Address, 0, len(ksAccounts))
	for _, account := range ksAccounts {
		addresses = append(addresses, account.Address)
	}

	return addresses, nil
}

func (m *DefaultManager) MigrateKeyStoreDir(newDir string, addresses []string) error {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}
	return m.keystore.MigrateKeyStoreDir(newDir, addresses)
}

func (m *DefaultManager) ReEncryptKeyStoreDir(oldPass, newPass string) error {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}
	return m.keystore.ReEncryptKeyStoreDir(oldPass, newPass)
}

func (m *DefaultManager) DeleteAccount(address ethtypes.Address) error {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}
	return m.keystore.Delete(address)
}

func (m *DefaultManager) GetVerifiedWalletAccount(db *accounts.Database, address ethtypes.Address, password string) (*generator.Account, error) {
	exists, err := db.AddressExists(address)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("account doesn't exist")
	}

	account, err := m.LoadAccount(address, password)
	if errors.Is(err, geth.ErrNoMatch) {
		account, err = m.generatePartialAccountKey(db, address, password)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	return account, nil
}

func (m *DefaultManager) generatePartialAccountKey(db *accounts.Database, address ethtypes.Address, password string) (*generator.Account, error) {
	rootAddress, err := db.GetWalletRootAddress()
	if err != nil {
		return nil, err
	}

	dbPath, err := db.GetPath(address)
	path := "m/" + dbPath[strings.LastIndex(dbPath, "/")+1:]
	if err != nil {
		return nil, err
	}

	return m.DeriveChildAccountForPathAndStore(rootAddress, path, password)
}

func (m *DefaultManager) DeriveChildAccountForPathAndStore(deriveFrom ethtypes.Address, path string, password string) (*generator.Account, error) {
	account, err := m.LoadAccount(deriveFrom, password)
	if err != nil {
		return nil, err
	}

	childAccount, err := generator.DeriveChildFromAccount(account, path)
	if err != nil {
		return nil, err
	}

	err = m.storeToKeystore(childAccount, password)
	if err != nil {
		return nil, err
	}

	return childAccount, nil
}

func (m *DefaultManager) DeriveChildrenAccountsForPathsAndStore(deriveFrom ethtypes.Address, paths []string, password string) (map[string]*generator.Account, error) {
	account, err := m.LoadAccount(deriveFrom, password)
	if err != nil {
		return nil, err
	}

	childAccounts, err := generator.DeriveChildrenFromAccount(account, paths)
	if err != nil {
		return nil, err
	}

	for _, childAccount := range childAccounts {
		err = m.storeToKeystore(childAccount, password)
		if err != nil {
			return nil, err
		}
	}

	return childAccounts, nil
}
