package accountsmanagement

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/accounts-management/keystore/geth"
	gocommon "github.com/status-im/status-go/common"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type ErrCannotLocateKeyFile struct {
	Msg string
}

func (e ErrCannotLocateKeyFile) Error() string {
	return e.Msg
}

// AccountsManager represents the default account manager implementation
type AccountsManager struct {
	keystoreMu  sync.RWMutex
	keystore    KeyStore
	persistence Persistence

	selectedChatAccountMutex sync.RWMutex
	selectedChatAccount      *generator.Account

	logger *zap.Logger
}

func NewAccountsManager(logger *zap.Logger) (*AccountsManager, error) {
	if logger == nil {
		return nil, ErrLoggerIsMissing
	}

	logger.Info("accounts manager created")
	return &AccountsManager{
		logger: logger,
	}, nil
}

func (m *AccountsManager) SetPersistence(persistence Persistence) {
	m.persistence = persistence
}

func (m *AccountsManager) SetKeystore(keystore KeyStore) {
	m.keystoreMu.Lock()
	defer m.keystoreMu.Unlock()

	m.keystore = keystore
	m.logger.Info("new keystore set")
}

// CreateAndStoreAccount creates an internal geth account and stores it in the keystore
func (m *AccountsManager) CreateAndStoreAccount(password string) (genAccount *generator.Account, mnemonic string, err error) {
	mnemonic, err = common.CreateRandomMnemonicWithDefaultLength()
	if err != nil {
		return
	}

	genAccount, err = m.CreateFromMnemonicAndStoreAccount(mnemonic, password)

	return
}

// CreateFromMnemonicAndStoreAccount creates an internal geth account from a mnemonic and stores it in the keystore
func (m *AccountsManager) CreateFromMnemonicAndStoreAccount(mnemonic string, password string) (genAccount *generator.Account, err error) {
	genAccount, err = generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return
	}

	err = m.storeToKeystore(genAccount, password)
	if err != nil {
		return
	}
	m.logger.Info("account created and stored (mnemonic)", zap.String("address", genAccount.Address().Hex()))

	return
}

func (m *AccountsManager) CreateFromPrivateKeyAndStoreAccount(privateKey string, password string) (genAccount *generator.Account, err error) {
	acc, err := generator.CreateAccountFromPrivateKey(privateKey)
	if err != nil {
		return
	}

	err = m.storeToKeystore(acc, password)
	if err != nil {
		return
	}
	m.logger.Info("account created and stored (private key)", zap.String("address", acc.Address().Hex()))

	return
}

// VerifyAccountPassword verifies if the account key for a given address and password is correct.
func (m *AccountsManager) VerifyAccountPassword(address ethtypes.Address, password string) (bool, error) {
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
func (m *AccountsManager) LoadAccount(address ethtypes.Address, password string) (*generator.Account, error) {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return nil, ErrAccountKeyStoreMissing
	}

	_, privateKey, extendedKey, err := m.keystore.AccountDecryptedKey(address, password)
	if err != nil {
		m.logger.Error("error loading account", zap.String("address", address.Hex()), zap.Error(err))
		if errors.Is(err, geth.ErrNoMatch) {
			return nil, &ErrCannotLocateKeyFile{fmt.Sprintf("cannot locate account for address: %s", address.Hex())}
		}
		return nil, err
	}

	account := generator.NewAccount(privateKey, extendedKey)
	if err := account.ValidateExtendedKey(); err != nil {
		m.logger.Error("error validating account", zap.String("address", address.Hex()), zap.Error(err))
		return nil, err
	}

	m.logger.Info("account loaded", zap.String("address", address.Hex()))
	return account, nil
}

func (m *AccountsManager) storeToKeystore(acc *generator.Account, password string) (err error) {
	m.keystoreMu.Lock()
	defer m.keystoreMu.Unlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}

	if acc == nil {
		return ErrAccountIsNil
	}

	m.logger.Info("storing account to keystore", zap.String("address", acc.Address().Hex()))

	if acc.HasExtendedKey() {
		_, err = m.keystore.ImportSingleExtendedKey(acc.ExtendedKey(), password)
		return
	}

	_, err = m.keystore.ImportECDSA(acc.PrivateKey(), password)
	return
}

func (m *AccountsManager) setChatAccount(account *generator.Account) {
	m.selectedChatAccountMutex.Lock()
	defer m.selectedChatAccountMutex.Unlock()

	m.selectedChatAccount = account
	if account != nil {
		m.logger.Info("chat account set", zap.String("address", account.Address().Hex()))
	} else {
		m.logger.Info("chat account cleared")
	}
}

// SetChatAccount sets the chat account with the given address and password
func (m *AccountsManager) SetChatAccount(chatAddress ethtypes.Address, password string) error {
	selectedChatAccount, err := m.LoadAccount(chatAddress, password)
	if err != nil {
		return err
	}

	m.setChatAccount(selectedChatAccount)

	return nil
}

// SetChatAccountWithPrivateKey sets the chat account with the given private key
func (m *AccountsManager) SetChatAccountWithPrivateKey(privKey *ecdsa.PrivateKey) error {
	account := generator.NewAccount(privKey, nil)

	m.setChatAccount(account)

	return nil
}

// SelectedChatAccount returns currently selected chat account
func (m *AccountsManager) SelectedChatAccount() (*generator.Account, error) {
	m.selectedChatAccountMutex.RLock()
	defer m.selectedChatAccountMutex.RUnlock()

	if m.selectedChatAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedChatAccount, nil
}

// Logout clears everything.
func (m *AccountsManager) Logout() {
	m.setChatAccount(nil)
	m.SetKeystore(nil)
	m.logger.Info("logout")
}

// Accounts returns list of addresses for selected account, including subaccounts.
func (m *AccountsManager) Accounts() ([]ethtypes.Address, error) {
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

func (m *AccountsManager) MigrateKeyStoreDir(newDir string, addresses []string) error {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}
	m.logger.Info("migrating keystore directory", zap.String("new location", newDir), zap.Strings("addresses", addresses))
	return m.keystore.MigrateKeyStoreDir(newDir, addresses)
}

func (m *AccountsManager) ReEncryptKeyStoreDir(oldPass, newPass string) error {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		return ErrAccountKeyStoreMissing
	}
	m.logger.Info("re-encrypting keystore directory")
	return m.keystore.ReEncryptKeyStoreDir(oldPass, newPass)
}

func (m *AccountsManager) DeleteAccount(address ethtypes.Address) error {
	m.keystoreMu.RLock()
	defer m.keystoreMu.RUnlock()

	if m.keystore == nil {
		m.logger.Error("cannot delete account, keystore is missing", zap.String("address", address.Hex()))
		return ErrAccountKeyStoreMissing
	}
	m.logger.Info("deleting account", zap.String("address", address.Hex()))
	return m.keystore.Delete(address)
}

func (m *AccountsManager) GetVerifiedWalletAccount(address ethtypes.Address, password string) (*generator.Account, error) {
	if m.persistence == nil {
		return nil, ErrPersistenceIsMissing
	}

	exists, err := m.persistence.AddressExists(address)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrAccountDoesNotExist
	}

	account, err := m.LoadAccount(address, password)
	if errors.Is(err, geth.ErrNoMatch) {
		account, err = m.generatePartialAccountKey(address, password)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	return account, nil
}

func (m *AccountsManager) generatePartialAccountKey(address ethtypes.Address, password string) (*generator.Account, error) {
	if m.persistence == nil {
		return nil, ErrPersistenceIsMissing
	}

	rootAddress, err := m.persistence.GetWalletRootAddress()
	if err != nil {
		return nil, err
	}

	dbPath, err := m.persistence.GetPath(address)
	if err != nil {
		return nil, err
	}
	path := "m/" + dbPath[strings.LastIndex(dbPath, "/")+1:]

	return m.DeriveChildAccountForPathAndStore(rootAddress, path, password)
}

func (m *AccountsManager) DeriveChildAccountForPathAndStore(deriveFrom ethtypes.Address, path string, password string) (*generator.Account, error) {
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

func (m *AccountsManager) DeriveChildrenAccountsForPathsAndStore(deriveFrom ethtypes.Address, paths []string, password string) (map[string]*generator.Account, error) {
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
