package accountsmanagement

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/accounts-management/keystore"
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

func (m *AccountsManager) createKeystore(keyUID string) (KeyStore, error) {
	// prepare keystore path
	const defaultKeystoreRelativePath = "keystore"
	relativePath := filepath.Join(defaultKeystoreRelativePath, keyUID)
	absoluteKeystorePath := filepath.Join(m.rootDataDir, relativePath)

	if _, err := os.Stat(absoluteKeystorePath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Clean(absoluteKeystorePath), os.ModePerm); err != nil {
			return nil, ErrKeystoreDirectoryError(err)
		}
	}

	return keystore.NewGethKeystoreAdapter(absoluteKeystorePath)
}

// deleteAccountFromKeystoreIfExists deletes an account from the keystore if it exists, if not returns no error
func (m *AccountsManager) deleteAccountFromKeystoreIfExists(address cryptotypes.Address, password string) error {
	if m.keystore == nil {
		m.logger.Error("cannot delete account, keystore is missing", zap.String("address", address.Hex()))
		return ErrKeystoreMissing
	}
	m.logger.Info("deleting account", zap.String("address", address.Hex()))
	err := m.keystore.Delete(address, password)
	if errors.Is(err, keystore.ErrKeystoreFileMissing) {
		return nil
	}
	return err
}

// deleteKeystoreFileForAccountIfExists deletes the keystore file for an account if it exists
// if the account is a watch only account, it does nothing
// if the account is non-operable or partially operable, it does nothing
// if the account belongs to regular, not keycard migrated keypair:
// - it deletes the keystore file for the account if the keystore file exists
// - if it is the last account of the keypair, it deletes the master account keystore file
func (m *AccountsManager) deleteKeystoreFileForAccountInternally(address cryptotypes.Address, password string) error {
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
		if password == "" {
			return ErrNoPasswordProvided
		}

		kp, err := m.persistence.GetKeypairByKeyUID(acc.KeyUID)
		if err != nil {
			return err
		}

		if !kp.MigratedToKeycard() {
			err = m.deleteAccountFromKeystoreIfExists(address, password)
			if err != nil {
				return err
			}

			if acc.Type != types.AccountTypeKey {
				lastAcccountOfKeypairWithTheSameKey := len(kp.Accounts) == 1
				if lastAcccountOfKeypairWithTheSameKey {
					err = m.deleteAccountFromKeystoreIfExists(cryptotypes.HexToAddress(kp.DerivedFrom), password)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// deleteKeystoreFilesForKeypairInternally deletes the keystore files for a keypair
// if the keypair is already migrated to keycard, it does nothing
// trying to delete a non-existent keystore file does not result in an error
func (m *AccountsManager) deleteKeystoreFilesForKeypairInternally(keypair *types.Keypair, password string) (err error) {
	if password == "" {
		return ErrNoPasswordProvided
	}

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
		err = m.deleteAccountFromKeystoreIfExists(acc.Address, password)
		if err != nil {
			return err
		}
	}

	if anyAccountFullyOrPartiallyOperable && keypair.Type != types.KeypairTypeKey {
		err = m.deleteAccountFromKeystoreIfExists(cryptotypes.HexToAddress(keypair.DerivedFrom), password)
		if err != nil {
			return err
		}
	}

	return
}

// CleanKeystoreFiles cleans the keystore files for all keypairs
// if the keypair is already migrated to keycard or removed, it cleans all accounts of the keypair, including the master account
// if the keypair is not migrated to keycard and not removed, it cleans the keystore files for removed accounts of the keypair,
// the master account is not cleaned if the keypair is not removed/migrated to keycard
func (m *AccountsManager) CleanKeystoreFiles(password string) error {
	if password == "" {
		return ErrNoPasswordProvided
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return ErrPersistenceMissing
	}

	keypairs, err := m.persistence.GetAllKeypairs()
	if err != nil {
		return err
	}

	for _, kp := range keypairs {
		if kp.MigratedToKeycard() || kp.Removed {
			for _, acc := range kp.Accounts {
				err = m.deleteAccountFromKeystoreIfExists(acc.Address, password)
				if err != nil {
					return err
				}
			}

			err = m.deleteAccountFromKeystoreIfExists(cryptotypes.HexToAddress(kp.DerivedFrom), password)
			if err != nil {
				return err
			}
			continue
		}

		for _, acc := range kp.Accounts {
			if !acc.Removed {
				continue
			}

			err = m.deleteAccountFromKeystoreIfExists(acc.Address, password)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
