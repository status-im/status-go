package core

import (
	"strings"

	"github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/accounts-management/keystore"
	"github.com/status-im/status-go/accounts-management/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

// CreateKeypairFromMnemonicAndStore creates a keypair with provided `walletAccount` and optionally a chat account if
// it's a profile keypair. It also stores the keypair/accounts to db and accounts to keystore if it's not a keycard.
// If it's a profile keypair, it also sets the chat account.
// Passed `walletAccount` is used to generate address using its path and set other details accordingly.
func (m *AccountsManager) CreateKeypairFromMnemonicAndStore(mnemonic string, password string, keypairName string,
	walletAccount *types.AccountCreationDetails, profile bool, clock uint64) (keypair *types.Keypair, err error) {

	if walletAccount == nil {
		err = ErrKeypairDoesNotHaveWalletAccount
		return
	}

	if !strings.HasPrefix(walletAccount.Path, common.WalletPath) {
		err = ErrUnsupportedWalletAccountPath.
			WithContext("path", walletAccount.Path).
			WithContext("expected path", common.WalletPath)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	paths := []string{
		walletAccount.Path,
	}
	if profile {
		paths = append(paths, common.PathEIP1581Chat)
	}

	// generate accounts from mnemonic
	var masterAccount *generator.Account
	var derivedAccounts map[string]*generator.Account
	masterAccount, derivedAccounts, err = generator.CreateAndDeriveAccountsFromMnemonic(mnemonic, paths, "")
	if err != nil {
		return
	}

	dbKeypair, err := m.persistence.GetKeypairByKeyUID(masterAccount.KeyUID())
	if err != nil {
		if err != types.ErrDbKeypairNotFound {
			return
		}
	}
	if dbKeypair != nil {
		err = ErrKeypairAlreadyAdded.WithContext("keyuid", masterAccount.KeyUID())
		return
	}

	keypairType := types.KeypairTypeSeed
	// make sure the keystore is created for the profile keypair
	if profile {
		keypairType = types.KeypairTypeProfile

		var keystore keystore.KeyStore
		keystore, err = m.createKeystore(masterAccount.KeyUID())
		if err != nil {
			return
		}
		m.setKeystore(keystore)
	}

	// prepare keypair
	keypair, err = m.prepareKeypair(masterAccount, derivedAccounts, keypairName, walletAccount, keypairType, profile, clock)
	if err != nil {
		return
	}

	// store keypair to db
	err = m.persistence.SaveOrUpdateKeypair(keypair)
	if err != nil {
		return
	}

	// store accounts to keystore
	err = m.storeKeystoreFilesForAccounts(masterAccount, derivedAccounts, password)
	if err != nil {
		return
	}

	// set the chat account if it's a profile keypair
	if profile {
		chatDerivedAccount, ok := derivedAccounts[common.PathEIP1581Chat]
		if !ok {
			return nil, ErrChatAccountNotFoundInDerivedAccounts
		}
		m.setChatAccountAndProfileKeyUID(chatDerivedAccount, masterAccount.KeyUID())
	}

	return
}

func (m *AccountsManager) AddKeypairStoredToKeycard(keyUID string, masterAddress string, name string,
	walletAccounts []*types.Account, clock uint64) (keypair *types.Keypair, err error) {

	if len(walletAccounts) == 0 {
		err = ErrKeypairMustHaveAtLeastOneWalletAccount
		return
	}

	for _, walletAccount := range walletAccounts {
		if !strings.HasPrefix(walletAccount.Path, common.WalletPath) {
			err = ErrUnsupportedWalletAccountPath.
				WithContext("path", walletAccount.Path).
				WithContext("expected path", common.WalletPath)
			return
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	var dbKeypair *types.Keypair
	dbKeypair, err = m.persistence.GetKeypairByKeyUID(keyUID)
	if err != nil {
		if err != types.ErrDbKeypairNotFound {
			return
		}
	}
	if dbKeypair != nil {
		err = ErrKeypairAlreadyAdded.WithContext("keyuid", keyUID)
		return
	}

	// prepare keypair
	keypair = &types.Keypair{
		Name:                    name,
		KeyUID:                  keyUID,
		Type:                    types.KeypairTypeSeed,
		DerivedFrom:             masterAddress,
		LastUsedDerivationIndex: 0,
		Clock:                   clock,
		Accounts:                walletAccounts,
	}

	// store keypair to db
	err = m.persistence.SaveOrUpdateKeypair(keypair)

	return
}

func (m *AccountsManager) prepareKeypair(account *generator.Account, derivedAccounts map[string]*generator.Account, keypairName string,
	walletAccount *types.AccountCreationDetails, keypairType types.KeypairType, profile bool, clock uint64) (*types.Keypair, error) {
	// set up keypair
	keypair := &types.Keypair{
		Name:                    keypairName,
		KeyUID:                  account.KeyUID(),
		Type:                    keypairType,
		DerivedFrom:             account.Address().Hex(),
		LastUsedDerivationIndex: 0,
		Clock:                   clock,
	}

	// add chat account
	chatDerivedAccount, ok := derivedAccounts[common.PathEIP1581Chat]
	if ok {
		keypair.Accounts = append(keypair.Accounts, &types.Account{
			PublicKey:          ethtypes.Hex2Bytes(chatDerivedAccount.PublicKeyHex()),
			KeyUID:             keypair.KeyUID,
			Address:            chatDerivedAccount.Address(),
			Chat:               profile,
			Path:               common.PathEIP1581Chat,
			AddressWasNotShown: true,
			Position:           -1,
			Operable:           types.AccountFullyOperable,
		})
	}

	position, err := m.persistence.GetPositionForNextNewAccount()
	if err != nil {
		return nil, err
	}

	// add wallet accounts
	walletDerivedAccount, ok := derivedAccounts[walletAccount.Path]
	if ok {
		keypair.Accounts = append(keypair.Accounts, &types.Account{
			PublicKey:          ethtypes.Hex2Bytes(walletDerivedAccount.PublicKeyHex()),
			KeyUID:             keypair.KeyUID,
			Address:            walletDerivedAccount.Address(),
			ColorID:            walletAccount.ColorID,
			Emoji:              walletAccount.Emoji,
			Wallet:             profile, // default wallet account makes sense only for profile keypair
			Path:               walletAccount.Path,
			Name:               walletAccount.Name,
			AddressWasNotShown: true,
			Position:           position,
			Operable:           types.AccountFullyOperable,
		})
	}

	if keypairType == types.KeypairTypeKey {
		keypair.DerivedFrom = ""

		keypair.Accounts = append(keypair.Accounts, &types.Account{
			PublicKey:          ethtypes.Hex2Bytes(account.PublicKeyHex()),
			KeyUID:             keypair.KeyUID,
			Address:            account.Address(),
			ColorID:            walletAccount.ColorID,
			Emoji:              walletAccount.Emoji,
			Path:               common.PathMaster,
			Name:               walletAccount.Name,
			AddressWasNotShown: true,
			Position:           position,
			Operable:           types.AccountFullyOperable,
		})
	}

	return keypair, nil
}

// CreateKeypairFromPrivateKeyAndStore creates a keypair with a single master account.
func (m *AccountsManager) CreateKeypairFromPrivateKeyAndStore(privateKey string, password string, keypairName string,
	walletAccount *types.AccountCreationDetails, clock uint64) (keypair *types.Keypair, err error) {
	if walletAccount == nil {
		err = ErrKeypairDoesNotHaveWalletAccount
		return
	}

	if walletAccount.Path != common.PathMaster {
		err = ErrUnsupportedWalletAccountPath.
			WithContext("path", walletAccount.Path).
			WithContext("expected path", common.PathMaster)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	masterAccount, err := generator.CreateAccountFromPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	dbKeypair, err := m.persistence.GetKeypairByKeyUID(masterAccount.KeyUID())
	if err != nil {
		if err != types.ErrDbKeypairNotFound {
			return
		}
	}
	if dbKeypair != nil {
		err = ErrKeypairAlreadyAdded.WithContext("keyuid", masterAccount.KeyUID())
		return
	}

	// prepare keypair
	keypair, err = m.prepareKeypair(masterAccount, nil, keypairName, walletAccount, types.KeypairTypeKey, false, clock)
	if err != nil {
		return nil, err
	}

	// store keypair to db
	err = m.persistence.SaveOrUpdateKeypair(keypair)
	if err != nil {
		return
	}

	// store the accounts to keystore, cause private key imported keypairs cannot be migrated to keycard
	err = m.storeKeystoreFilesForAccounts(masterAccount, nil, password)
	if err != nil {
		return
	}

	return keypair, nil
}

// MakeSeedPhraseKeypairFullyOperable checks if corresponding keypair exists in db and if yes, creates the keystore files
// for the keypair and associated accounts and marks the keypair as fully operable
func (m *AccountsManager) MakeSeedPhraseKeypairFullyOperable(mnemonic string, password string, clock uint64) (string, error) {
	acc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return "", err
	}

	keyUID := acc.KeyUID()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return "", ErrPersistenceMissing
	}

	kp, err := m.persistence.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return "", err
	}

	var paths []string
	for _, acc := range kp.Accounts {
		paths = append(paths, acc.Path)
	}

	_, _, err = m.storeKeystoreFilesForMnemonicInternally(mnemonic, password, paths)
	if err != nil {
		return "", err
	}

	return keyUID, m.persistence.MarkKeypairFullyOperable(keyUID, clock, true)
}

// MakePrivateKeyKeypairFullyOperable checks if corresponding keypair exists in db and if yes, creates the keystore file for the keypair
func (m *AccountsManager) MakePrivateKeyKeypairFullyOperable(privateKey string, password string, clock uint64) (string, error) {
	acc, err := generator.CreateAccountFromPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return "", ErrPersistenceMissing
	}

	kp, err := m.persistence.GetKeypairByKeyUID(acc.KeyUID())
	if err != nil {
		return "", err
	}

	_, err = m.storeKeystoreFilesForPrivateKeyInternally(privateKey, password)
	if err != nil {
		return "", err
	}

	return kp.KeyUID, m.persistence.MarkKeypairFullyOperable(kp.KeyUID, clock, true)
}

func (m *AccountsManager) MakePartiallyOperableAccoutsFullyOperable(password string) (addresses []ethtypes.Address, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	keypairs, err := m.persistence.GetActiveKeypairs()
	if err != nil {
		return
	}

	for _, kp := range keypairs {
		if kp.MigratedToKeycard() {
			continue
		}

		for _, acc := range kp.Accounts {
			if acc.Operable != types.AccountPartiallyOperable {
				continue
			}
			_, err = m.deriveChildAccountForPathAndStore(ethtypes.HexToAddress(kp.DerivedFrom), acc.Path, password)
			if err != nil {
				return
			}
			err = m.persistence.MarkAccountFullyOperable(acc.Address)
			if err != nil {
				return
			}
			addresses = append(addresses, acc.Address)
		}
	}
	return
}

func (m *AccountsManager) AddAccounts(keyUID string, accounts []*types.Account, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return ErrPersistenceMissing
	}

	kp, err := m.persistence.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return err
	}

	if kp.Type == types.KeypairTypeKey {
		return ErrCannotAddAccountsToKeypairImportedViaPrivateKey
	}

	if !kp.MigratedToKeycard() {
		for _, acc := range accounts {
			if acc.KeyUID != keyUID {
				return ErrAccountMismatch.
					WithContext("keyuid", acc.KeyUID).
					WithContext("expected keyuid", keyUID)
			}

			if kp.Type == types.KeypairTypeProfile {
				if acc.Chat {
					return ErrCannotAddDefaultChatAccount
				}
				if acc.Wallet {
					return ErrCannotAddDefaultWalletAccount
				}
			}

			for _, kpAcc := range kp.Accounts {
				if acc.Address == kpAcc.Address {
					return ErrAccountAlreadyAdded
				}
			}

			childAccount, err := m.deriveChildAccountForPath(ethtypes.HexToAddress(kp.DerivedFrom), acc.Path, password)
			if err != nil {
				return err
			}

			if childAccount.Address() != acc.Address {
				return ErrAccountMismatch.
					WithContext("address", acc.Address.Hex()).
					WithContext("derived address", childAccount.Address().Hex())
			}
		}
	}

	err = m.persistence.SaveOrUpdateAccounts(accounts, true)
	if err != nil {
		return err
	}

	if !kp.MigratedToKeycard() {
		for _, acc := range accounts {
			_, err := m.deriveChildAccountForPathAndStore(ethtypes.HexToAddress(kp.DerivedFrom), acc.Path, password)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *AccountsManager) RemoveAccounts(keyUID string, accounts []*types.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return ErrPersistenceMissing
	}

	kp, err := m.persistence.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return err
	}

	if !kp.MigratedToKeycard() {
		for _, acc := range accounts {
			err = m.deleteAccountFromKeystore(acc.Address)
			if err != nil {
				return err
			}
		}
	}

	return m.persistence.SaveOrUpdateAccounts(accounts, true)

}

func (m *AccountsManager) MigrateNonProfileKeycardKeypairToApp(mnemonic string, password string, clock uint64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return "", ErrPersistenceMissing
	}

	acc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return "", err
	}

	kp, err := m.persistence.GetKeypairByKeyUID(acc.KeyUID())
	if err != nil {
		return "", err
	}

	if kp.Type == types.KeypairTypeProfile {
		return "", ErrCannotMigrateProfileKeypair
	}

	if !kp.MigratedToKeycard() {
		return "", ErrKeypairIsNotKeycard
	}

	profileKeypair, err := m.persistence.GetProfileKeypair()
	if err != nil {
		return "", err
	}

	if !profileKeypair.MigratedToKeycard() {
		_, err = m.loadAccountInternally(ethtypes.HexToAddress(profileKeypair.DerivedFrom), password)
		if err != nil {
			return "", ErrWrongPasswordProvided(err)
		}
	}

	var paths []string
	for _, acc := range kp.Accounts {
		paths = append(paths, acc.Path)
	}

	_, _, err = m.storeKeystoreFilesForMnemonicInternally(mnemonic, password, paths)
	if err != nil {
		return "", err
	}

	return kp.KeyUID, m.persistence.DeleteAllKeycardsWithKeyUID(kp.KeyUID, clock)
}

// SaveOrUpdateKeycard saves or updates a keycard and its accounts
// If `removeKeystoreFiles` is true, then corresponding keystore files will be deleted if keypair is not already migrated.
// In case of migration to a keycard, corresponding keystore files need to be deleted if the keypair being migrated is not already migrated.
func (m *AccountsManager) SaveOrUpdateKeycard(keycard *types.Keycard, clock uint64, removeKeystoreFiles bool) error {
	if len(keycard.AccountsAddresses) == 0 {
		return ErrKeycardDoesNotHaveAnyAccounts
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return ErrPersistenceMissing
	}

	kpDb, err := m.persistence.GetKeypairByKeyUID(keycard.KeyUID)
	if err != nil {
		return ErrKeycardDoesNotRelateToAnyKeypair(err)
	}

	err = m.persistence.SaveOrUpdateKeycard(keycard, clock, true)
	if err != nil {
		return err
	}

	if removeKeystoreFiles {
		// Once we migrate a keypair, corresponding keystore files need to be deleted
		// if the keypair being migrated is not already migrated (in case user is creating a copy of an existing Keycard)
		// and if keypair operability is different from non operable (otherwise there are not keystore files to be deleted).
		if !kpDb.MigratedToKeycard() && kpDb.Operability() != types.AccountNonOperable {
			for _, acc := range kpDb.Accounts {
				if acc.Operable != types.AccountFullyOperable {
					continue
				}
				err = m.deleteAccountFromKeystore(acc.Address)
				if err != nil {
					return err
				}
			}

			err = m.deleteAccountFromKeystore(ethtypes.HexToAddress(kpDb.DerivedFrom))
			if err != nil {
				return err
			}
		}

		err = m.persistence.MarkKeypairFullyOperable(keycard.KeyUID, clock, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *AccountsManager) DeleteAccount(address ethtypes.Address, clock uint64) (account *types.Account, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	account, err = m.persistence.GetAccountByAddress(address)
	if err != nil {
		return
	}

	if account.Chat {
		err = ErrCannotRemoveDefaultChatAccount
		return
	}

	if account.Wallet {
		err = ErrCannotRemoveDefaultWalletAccount
		return
	}

	err = m.deleteKeystoreFileForAccountInternally(address)
	if err != nil {
		return
	}

	err = m.persistence.RemoveAccount(address, clock)
	return
}

func (m *AccountsManager) DeleteKeypair(keyUID string, clock uint64) (keypair *types.Keypair, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.persistence == nil {
		return nil, ErrPersistenceMissing
	}

	keypair, err = m.persistence.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return
	}

	if keypair.Type == types.KeypairTypeProfile {
		err = ErrCannotRemoveProfileKeypair
		return
	}

	err = m.deleteKeystoreFilesForKeypairInternally(keypair)
	if err != nil {
		return
	}

	err = m.persistence.RemoveKeypair(keyUID, clock)
	return
}
