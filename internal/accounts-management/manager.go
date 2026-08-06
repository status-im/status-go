package accountsmanagement

import (
	"crypto/ecdsa"
	"errors"
	"sync"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	accsmanagementerrors "github.com/status-im/status-go/internal/accounts-management/errors"
	"github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// SecretResolver translates the password provided by the client into the secret
type SecretResolver func(password string) (string, error)

// AccountsManager represents the default account manager implementation
type AccountsManager struct {
	mu          sync.RWMutex
	keystore    KeyStore
	persistence Persistence

	rootDataDir         string
	profileKeyUID       string
	selectedChatAccount *generator.Account
	secretResolver      SecretResolver

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

// SetPersistence sets the persistence for the accounts manager
func (m *AccountsManager) SetPersistence(persistence Persistence) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.persistence = persistence
}

// SetRootDataDir sets the root data directory for the accounts manager
func (m *AccountsManager) SetRootDataDir(rootDataDir string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rootDataDir = rootDataDir
}

// SetSecretResolver sets (or clears, with nil) the resolver
func (m *AccountsManager) SetSecretResolver(resolver SecretResolver) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.secretResolver = resolver
}

// resolveSecret applies the configured secret resolver to a client-provided password
// Callers must hold m.mu (read or write).
func (m *AccountsManager) resolveSecret(password string) (string, error) {
	if m.secretResolver == nil {
		return password, nil
	}
	return m.secretResolver(password)
}

func (m *AccountsManager) setKeystore(keystore KeyStore) {
	m.keystore = keystore
}

func (m *AccountsManager) setChatAccountAndProfileKeyUID(account *generator.Account, profileKeyUID string) {
	m.selectedChatAccount = account
	m.profileKeyUID = profileKeyUID
}

func (m *AccountsManager) isChatAccountSet(address cryptotypes.Address) bool {
	return m.selectedChatAccount != nil && m.selectedChatAccount.Address() == address
}

// VerifyAccountPassword verifies if the account key for a given address and password is correct.
func (m *AccountsManager) VerifyAccountPassword(address cryptotypes.Address, password string) (bool, error) {
	account, err := m.LoadAccount(address, password)
	if err != nil {
		return false, err
	}

	if account.Address() != address {
		return false, ErrAccountMismatch.
			WithContext("got", gocommon.TruncateWithDot(account.Address().Hex())).
			WithContext("want", gocommon.TruncateWithDot(address.Hex()))
	}

	return true, nil
}

// LoadAccount loads an account key from the keystore for a given address and password.
// If either address or password is incorrect, an error is returned.
func (m *AccountsManager) LoadAccount(address cryptotypes.Address, password string) (*generator.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loadAccountInternally(address, password)
}

func (m *AccountsManager) loadAccountInternally(address cryptotypes.Address, password string) (*generator.Account, error) {
	if m.keystore == nil {
		return nil, ErrKeystoreMissing
	}

	secret, err := m.resolveSecret(password)
	if err != nil {
		return nil, err
	}

	_, privateKey, extendedKey, err := m.keystore.AccountDecryptedKey(address, secret)
	if err != nil && secret != password {
		// Fallback for interrupted migrations.
		if _, fbPrivateKey, fbExtendedKey, fbErr := m.keystore.AccountDecryptedKey(address, password); fbErr == nil {
			m.logger.Warn("keystore file decrypted with legacy credentials after failed DEK decrypt; encryption-scheme migration was likely interrupted",
				zap.String("address", address.Hex()))
			privateKey, extendedKey, err = fbPrivateKey, fbExtendedKey, nil
		}
	}
	if err != nil {
		m.logger.Error("error loading account", zap.String("address", address.Hex()), zap.Error(err))
		if errors.Is(err, keystore.ErrKeystoreFileMissing) {
			m.logger.Error("cannot locate account for address", zap.String("address", address.Hex()))
			return nil, keystore.ErrKeystoreFileMissing.Copy().WithContext("address", address.Hex())
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

func (m *AccountsManager) loadPrivateKeyInternally(address cryptotypes.Address, password string) (*generator.Account, error) {
	if m.keystore == nil {
		return nil, ErrKeystoreMissing
	}

	secret, err := m.resolveSecret(password)
	if err != nil {
		return nil, err
	}

	_, privateKey, err := m.keystore.AccountDecryptedPrivateKey(address, secret)
	if err != nil && secret != password {
		// Fallback for interrupted migrations.
		if _, fbPrivateKey, fbErr := m.keystore.AccountDecryptedPrivateKey(address, password); fbErr == nil {
			m.logger.Warn("keystore file decrypted with legacy credentials after failed DEK decrypt; encryption-scheme migration was likely interrupted",
				zap.String("address", address.Hex()))
			privateKey, err = fbPrivateKey, nil
		}
	}
	if err != nil {
		m.logger.Error("error loading account", zap.String("address", address.Hex()), zap.Error(err))
		if errors.Is(err, keystore.ErrKeystoreFileMissing) {
			m.logger.Error("cannot locate account for address", zap.String("address", address.Hex()))
			return nil, keystore.ErrKeystoreFileMissing.Copy().WithContext("address", address.Hex())
		}
		return nil, err
	}

	return generator.NewAccount(privateKey, nil), nil
}

// SetChatAccount sets the chat account and keystore either by address and password or by private key
func (m *AccountsManager) SetChatAccount(address cryptotypes.Address, password string, privateKey *ecdsa.PrivateKey) error {
	return m.setChatAccount(address, password, privateKey, nil)
}

func (m *AccountsManager) SetChatAccountWithProfileKeypair(address cryptotypes.Address, password string, privateKey *ecdsa.PrivateKey, profileKeypair *accsmanagementtypes.Keypair) error {
	return m.setChatAccount(address, password, privateKey, profileKeypair)
}

func (m *AccountsManager) setChatAccount(address cryptotypes.Address, password string, privateKey *ecdsa.PrivateKey, profileKeypair *accsmanagementtypes.Keypair) error {
	if address == cryptotypes.ZeroAddress() && privateKey == nil {
		return ErrAddressAndPasswordOrPrivateKeyRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isChatAccountSet(address) {
		m.logger.Info("chat account already set", zap.String("address", address.Hex()))
		return nil
	}

	var err error
	if profileKeypair == nil {
		profileKeypair, err = m.persistence.GetProfileKeypair()
	}
	if err != nil {
		return err
	}

	keystore, err := m.createKeystore(profileKeypair.KeyUID)
	if err != nil {
		return err
	}
	m.setKeystore(keystore)

	var selectedChatAccount *generator.Account
	if privateKey != nil {
		selectedChatAccount = generator.NewAccount(privateKey, nil)
	} else {
		selectedChatAccount, err = m.loadPrivateKeyInternally(address, password)
		if err != nil {
			return err
		}
	}

	m.setChatAccountAndProfileKeyUID(selectedChatAccount, profileKeypair.KeyUID)

	return nil
}

// SelectedChatAccount returns currently selected chat account
func (m *AccountsManager) SelectedChatAccount() (*generator.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.selectedChatAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedChatAccount, nil
}

// Logout clears everything.
func (m *AccountsManager) Logout() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keystore = nil
	m.persistence = nil
	m.rootDataDir = ""
	m.selectedChatAccount = nil
	m.profileKeyUID = ""
	m.secretResolver = nil
	m.logger.Info("logout")
}

// Accounts returns list of addresses for selected account, including subaccounts.
func (m *AccountsManager) Accounts() ([]cryptotypes.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.keystore == nil {
		return nil, ErrKeystoreMissing
	}

	ksAccounts := m.keystore.Accounts()
	addresses := make([]cryptotypes.Address, 0, len(ksAccounts))
	for _, account := range ksAccounts {
		addresses = append(addresses, account.Address)
	}

	return addresses, nil
}

// GetVerifiedWalletAccount gets a verified wallet account by address and password.
// If the account has no its own keystore file (partially operable account), then its key is derived from the master keystore file and stored.
func (m *AccountsManager) GetVerifiedWalletAccount(address cryptotypes.Address, password string) (*generator.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	exists, err := m.persistence.AddressExists(address)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrAccountDoesNotExist.WithContext("address", address.Hex())
	}

	account, err := m.loadAccountInternally(address, password)
	if err != nil {
		var accountsErr *accsmanagementerrors.AccountsError
		if errors.As(err, &accountsErr) && accountsErr.Is(keystore.ErrKeystoreFileMissing) {
			account, err = m.generatePartialAccountKey(address, password)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return account, nil
}
