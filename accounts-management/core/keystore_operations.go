package core

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/accounts-management/keystore"
	"github.com/status-im/status-go/accounts-management/keystore/geth"
	"github.com/status-im/status-go/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/crypto/types"
)

// ReloadKeystore reloads the keystore for the selected chat account
func (m *AccountsManager) ReloadKeystore() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.selectedChatAccount == nil || m.profileKeyUID == "" {
		return ErrNoAccountSelected
	}

	keystore, err := m.createKeystore(m.profileKeyUID)
	if err != nil {
		return err
	}
	m.setKeystore(keystore)
	return nil
}

// StoreKeystoreFilesForMnemonic stores the keystore files for an account created from a given mnemonic and for all derived accounts from given paths
func (m *AccountsManager) StoreKeystoreFilesForMnemonic(mnemonic string, password string, paths []string) (account *generator.Account, derivedAccounts map[string]*generator.Account, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.storeKeystoreFilesForMnemonicInternally(mnemonic, password, paths)
}

func (m *AccountsManager) storeKeystoreFilesForMnemonicInternally(mnemonic string, password string, paths []string) (account *generator.Account, derivedAccounts map[string]*generator.Account, err error) {
	account, derivedAccounts, err = generator.CreateAndDeriveAccountsFromMnemonic(mnemonic, paths, "")
	if err != nil {
		return
	}

	err = m.storeKeystoreFilesForAccounts(account, derivedAccounts, password)
	if err != nil {
		return
	}

	return
}

// StoreKeystoreFilesForPrivateKey stores the keystore file for an account created from a given private key
func (m *AccountsManager) StoreKeystoreFilesForPrivateKey(privateKey string, password string) (account *generator.Account, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.storeKeystoreFilesForPrivateKeyInternally(privateKey, password)
}

func (m *AccountsManager) storeKeystoreFilesForPrivateKeyInternally(privateKey string, password string) (account *generator.Account, err error) {

	account, err = generator.CreateAccountFromPrivateKey(privateKey)
	if err != nil {
		return
	}

	err = m.storeKeystoreFilesForAccounts(account, nil, password)
	if err != nil {
		return
	}

	return
}

func (m *AccountsManager) storeKeystoreFilesForAccounts(account *generator.Account, derivedAccounts map[string]*generator.Account, password string) (err error) {
	err = m.storeToKeystore(account, password)
	if err != nil {
		return
	}

	m.logger.Info("master account stored (mnemonic)", zap.String("address", account.Address().Hex()))

	for path, acc := range derivedAccounts {
		err = m.storeToKeystore(acc, password)
		if err != nil {
			return
		}
		m.logger.Info("account on path created and stored (mnemonic)", zap.String("path", path), zap.String("address", acc.Address().Hex()))
	}

	return
}

func (m *AccountsManager) storeToKeystore(acc *generator.Account, password string) (err error) {
	if m.keystore == nil {
		return ErrKeystoreMissing
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

func (m *AccountsManager) createKeystore(keyUID string) (keystore.KeyStore, error) {
	// prepare keystore path
	const DefaultKeystoreRelativePath = "keystore"
	relativePath := filepath.Join(DefaultKeystoreRelativePath, keyUID)
	absoluteKeystorePath := filepath.Join(m.rootDataDir, relativePath)

	if _, err := os.Stat(absoluteKeystorePath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Clean(absoluteKeystorePath), os.ModePerm); err != nil {
			return nil, ErrKeystoreDirectoryError(err)
		}
	}

	return geth.NewGethKeystoreAdapter(absoluteKeystorePath)
}

// deleteAccountFromKeystore deletes an account from the keystore
func (m *AccountsManager) deleteAccountFromKeystore(address cryptotypes.Address) error {
	if m.keystore == nil {
		m.logger.Error("cannot delete account, keystore is missing", zap.String("address", address.Hex()))
		return ErrKeystoreMissing
	}
	m.logger.Info("deleting account", zap.String("address", address.Hex()))
	return m.keystore.Delete(address)
}

// DeleteKeystoreFileForAccount deletes the keystore file for an account
// if the account is non-operable or partially operable, it does nothing
// if the account is a watch account, it does nothing
// if the account is a key account, it deletes the keystore file for the account
// if the account is a key account and it is the last account of the keypair, it deletes the master account keystore file
// trying to delete a non-existent keystore file for an account does not result in an error
func (m *AccountsManager) DeleteKeystoreFileForAccount(address cryptotypes.Address) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.deleteKeystoreFileForAccountInternally(address)
}

func (m *AccountsManager) deleteKeystoreFileForAccountInternally(address cryptotypes.Address) error {
	if m.persistence == nil {
		return ErrPersistenceMissing
	}

	acc, err := m.persistence.GetAccountByAddress(address)
	if err != nil {
		return err
	}

	if acc.Operable == types.AccountNonOperable || acc.Operable == types.AccountPartiallyOperable {
		return nil
	}

	if acc.Type != types.AccountTypeWatch {
		kp, err := m.persistence.GetKeypairByKeyUID(acc.KeyUID)
		if err != nil {
			return err
		}

		if !kp.MigratedToKeycard() {
			err = m.deleteAccountFromKeystore(address)
			if err != nil {
				return err
			}

			if acc.Type != types.AccountTypeKey {
				lastAcccountOfKeypairWithTheSameKey := len(kp.Accounts) == 1
				if lastAcccountOfKeypairWithTheSameKey {
					err = m.deleteAccountFromKeystore(cryptotypes.HexToAddress(kp.DerivedFrom))
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// DeleteKeystoreFilesForKeypair deletes the keystore files for a keypair
// if the keypair is already migrated to keycard, it does nothing
// trying to delete a non-existent keystore file does not result in an error
func (m *AccountsManager) DeleteKeystoreFilesForKeypair(keypair *types.Keypair) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.deleteKeystoreFilesForKeypairInternally(keypair)
}

func (m *AccountsManager) deleteKeystoreFilesForKeypairInternally(keypair *types.Keypair) (err error) {
	if keypair == nil {
		return ErrKeypairIsNil
	}

	if keypair.MigratedToKeycard() {
		return nil
	}

	if m.persistence == nil {
		return ErrPersistenceMissing
	}

	anyAccountFullyOrPartiallyOperable := false
	for _, acc := range keypair.Accounts {
		if acc.Removed || acc.Operable == types.AccountNonOperable {
			continue
		}
		if !anyAccountFullyOrPartiallyOperable {
			anyAccountFullyOrPartiallyOperable = true
		}
		if acc.Operable == types.AccountPartiallyOperable {
			continue
		}
		err = m.deleteAccountFromKeystore(acc.Address)
		if err != nil {
			return err
		}
	}

	if anyAccountFullyOrPartiallyOperable && keypair.Type != types.KeypairTypeKey {
		err = m.deleteAccountFromKeystore(cryptotypes.HexToAddress(keypair.DerivedFrom))
		if err != nil {
			return err
		}
	}

	return
}

// MigrateKeyStoreDir migrates the keystore directory from the current location to the provided new location
func (m *AccountsManager) MigrateKeyStoreDir(newDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.keystore == nil {
		return ErrKeystoreMissing
	}
	m.logger.Info("migrating keystore directory", zap.String("new location", newDir))
	return m.keystore.MigrateKeyStoreDir(newDir)
}

// ReEncryptKeyStoreDir re-encrypts the keystore directory with the provided old and new passwords
func (m *AccountsManager) ReEncryptKeyStoreDir(oldPass, newPass string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.keystore == nil {
		return ErrKeystoreMissing
	}
	m.logger.Info("re-encrypting keystore directory")
	return m.keystore.ReEncryptKeyStoreDir(oldPass, newPass)
}

func (m *AccountsManager) generatePartialAccountKey(address cryptotypes.Address, password string) (*generator.Account, error) {
	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	rootAddress, err := m.persistence.GetWalletRootAddress()
	if err != nil {
		return nil, err
	}

	acc, err := m.persistence.GetAccountByAddress(address)
	if err != nil {
		return nil, err
	}

	dbPath, err := m.persistence.GetPath(acc.Address)
	if err != nil {
		return nil, err
	}
	path := "m/" + dbPath[strings.LastIndex(dbPath, "/")+1:]

	return m.deriveChildAccountForPathAndStore(rootAddress, path, password)
}

func (m *AccountsManager) deriveChildAccountForPath(deriveFrom cryptotypes.Address, path string, password string) (*generator.Account, error) {
	account, err := m.loadAccountInternally(deriveFrom, password)
	if err != nil {
		return nil, err
	}

	return generator.DeriveChildFromAccount(account, path)
}

func (m *AccountsManager) deriveChildAccountForPathAndStore(deriveFrom cryptotypes.Address, path string, password string) (*generator.Account, error) {
	childAccount, err := m.deriveChildAccountForPath(deriveFrom, path, password)
	if err != nil {
		return nil, err
	}

	err = m.storeToKeystore(childAccount, password)
	if err != nil {
		return nil, err
	}

	return childAccount, nil
}
