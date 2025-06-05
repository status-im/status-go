package account

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/status-im/extkeys"

	gethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account/common"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/account/types"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/keystore"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

type ErrCannotLocateKeyFile struct {
	Msg string
}

func (e ErrCannotLocateKeyFile) Error() string {
	return e.Msg
}

// Manager represents account manager interface
type Manager interface {
	GetVerifiedWalletAccount(db *accounts.Database, address, password string) (*types.SelectedExtKey, error)
	DeleteAccount(address ethtypes.Address) error
}

// DefaultManager represents default account manager implementation
type DefaultManager struct {
	mu       sync.RWMutex
	Keydir   string
	keystore types.KeyStore

	accountsGenerator *generator.Generator
	onboarding        *Onboarding

	selectedChatAccount *types.SelectedExtKey // account that was processed during the last call to SelectAccount()
	mainAccountAddress  ethtypes.Address
	watchAddresses      []ethtypes.Address

	logger *zap.Logger
}

// AccountsGenerator returns accountsGenerator.
func (m *DefaultManager) AccountsGenerator() *generator.Generator {
	return m.accountsGenerator
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (m *DefaultManager) CreateAccount(password string) (generator.GeneratedAccountInfo, types.Info, string, error) {
	var mkInfo generator.GeneratedAccountInfo
	info := types.Info{}

	// generate mnemonic phrase
	mn := extkeys.NewMnemonic()
	mnemonic, err := mn.MnemonicPhrase(extkeys.EntropyStrength128, extkeys.EnglishLanguage)
	if err != nil {
		return mkInfo, info, "", fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	extendedKey, err := common.CreateExtendedKeyFromMnemonic(mnemonic, "")
	if err != nil {
		return mkInfo, info, "", err
	}

	acc := generator.NewAccount(nil, extendedKey)
	mkInfo = acc.ToGeneratedAccountInfo("", mnemonic)

	// import created key into account keystore
	info.WalletAddress, info.WalletPubKey, err = m.importExtendedKey(extkeys.KeyPurposeWallet, extendedKey, password)
	if err != nil {
		return mkInfo, info, "", err
	}

	info.ChatAddress = info.WalletAddress
	info.ChatPubKey = info.WalletPubKey

	return mkInfo, info, mnemonic, nil
}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (m *DefaultManager) RecoverAccount(password, mnemonic string) (types.Info, error) {
	info := types.Info{}
	// re-create extended key (see BIP32)
	extendedKey, err := common.CreateExtendedKeyFromMnemonic(mnemonic, "")
	if err != nil {
		return info, err
	}

	// import re-created key into account keystore
	info.WalletAddress, info.WalletPubKey, err = m.importExtendedKey(extkeys.KeyPurposeWallet, extendedKey, password)
	if err != nil {
		return info, err
	}

	info.ChatAddress = info.WalletAddress
	info.ChatPubKey = info.WalletPubKey

	return info, nil
}

// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
// If no error is returned, then account is considered verified.
func (m *DefaultManager) VerifyAccountPassword(keyStoreDir, address, password string) (*ethtypes.Key, error) {
	var err error
	var foundKeyFile []byte

	addressObj := ethtypes.BytesToAddress(ethtypes.FromHex(address))
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
		if ethtypes.HexToAddress("0x"+accountKey.Address).Hex() == addressObj.Hex() {
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
		return nil, &ErrCannotLocateKeyFile{fmt.Sprintf("cannot locate account for address: %s", addressObj.Hex())}
	}

	key, err := keystore.DecryptKey(foundKeyFile, password)
	if err != nil {
		return nil, err
	}

	// avoid swap attack
	if key.Address != addressObj {
		return nil, fmt.Errorf("account mismatch: have %s, want %s", gocommon.TruncateWithDot(key.Address.Hex()), gocommon.TruncateWithDot(addressObj.Hex()))
	}

	return key, nil
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, all previous identities are removed).
func (m *DefaultManager) SelectAccount(loginParams types.LoginParams) error {
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

func (m *DefaultManager) SetAccountAddresses(main ethtypes.Address, secondary ...ethtypes.Address) {
	m.watchAddresses = []ethtypes.Address{main}
	m.watchAddresses = append(m.watchAddresses, secondary...)
	m.mainAccountAddress = main
}

// SetChatAccount initializes selectedChatAccount with privKey
func (m *DefaultManager) SetChatAccount(privKey *ecdsa.PrivateKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	address := crypto.PubkeyToAddress(privKey.PublicKey)
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	key := &ethtypes.Key{
		ID:         id,
		Address:    address,
		PrivateKey: privKey,
	}

	m.selectedChatAccount = &types.SelectedExtKey{
		Address:    address,
		AccountKey: key,
	}
	return nil
}

// MainAccountAddress returns main account address set during login
func (m *DefaultManager) MainAccountAddress() (ethtypes.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.mainAccountAddress == ethtypes.ZeroAddress() {
		return ethtypes.ZeroAddress(), ErrNoAccountSelected
	}

	return m.mainAccountAddress, nil
}

// WatchAddresses returns currently selected watch addresses.
func (m *DefaultManager) WatchAddresses() []ethtypes.Address {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.watchAddresses
}

// SelectedChatAccount returns currently selected chat account
func (m *DefaultManager) SelectedChatAccount() (*types.SelectedExtKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.selectedChatAccount == nil {
		return nil, ErrNoAccountSelected
	}
	return m.selectedChatAccount, nil
}

// Logout clears selected accounts.
func (m *DefaultManager) Logout() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.accountsGenerator.Reset()
	m.mainAccountAddress = ethtypes.ZeroAddress()
	m.watchAddresses = nil
	m.selectedChatAccount = nil
}

// ImportAccount imports the account specified with privateKey.
func (m *DefaultManager) ImportAccount(privateKey *ecdsa.PrivateKey, password string) (ethtypes.Address, error) {
	if m.keystore == nil {
		return ethtypes.Address{}, ErrAccountKeyStoreMissing
	}

	account, err := m.keystore.ImportECDSA(privateKey, password)

	return account.Address, err
}

// ImportSingleExtendedKey imports an extended key setting it in both the PrivateKey and ExtendedKey fields
// of the Key struct.
// ImportExtendedKey is used in older version of Status where PrivateKey is set to be the BIP44 key at index 0,
// and ExtendedKey is the extended key of the BIP44 key at index 1.
func (m *DefaultManager) ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
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

	pubKey = ethtypes.EncodeHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}

// importExtendedKey processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
func (m *DefaultManager) importExtendedKey(keyPurpose extkeys.KeyPurpose, extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
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
	pubKey = ethtypes.EncodeHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}

// Accounts returns list of addresses for selected account, including
// subaccounts.
func (m *DefaultManager) Accounts() ([]ethtypes.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	addresses := make([]ethtypes.Address, 0)
	if m.mainAccountAddress != ethtypes.ZeroAddress() {
		addresses = append(addresses, m.mainAccountAddress)
	}

	return addresses, nil
}

// StartOnboarding starts the onboarding process generating accountsCount accounts and returns a slice of OnboardingAccount.
func (m *DefaultManager) StartOnboarding(accountsCount, mnemonicPhraseLength int) ([]*OnboardingAccount, error) {
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
func (m *DefaultManager) RemoveOnboarding() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onboarding = nil
}

// ImportOnboardingAccount imports the account specified by id and encrypts it with password.
func (m *DefaultManager) ImportOnboardingAccount(id string, password string) (types.Info, string, error) {
	var info types.Info

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
func (m *DefaultManager) AddressToDecryptedAccount(address, password string) (types.Account, *ethtypes.Key, error) {
	if m.keystore == nil {
		return types.Account{}, nil, ErrAccountKeyStoreMissing
	}

	account, err := types.AddressToAccount(address)
	if err != nil {
		return types.Account{}, nil, ErrAddressToAccountMappingFailure
	}

	account, key, err := m.keystore.AccountDecryptedKey(account, password)
	if err != nil {
		err = fmt.Errorf("%s: %s", ErrAccountToKeyMappingFailure, err)
	}

	return account, key, err
}

func (m *DefaultManager) unlockExtendedKey(address, password string) (*types.SelectedExtKey, error) {
	account, accountKey, err := m.AddressToDecryptedAccount(address, password)
	if err != nil {
		return nil, err
	}

	selectedExtendedKey := &types.SelectedExtKey{
		Address:    account.Address,
		AccountKey: accountKey,
	}

	return selectedExtendedKey, nil
}

func (m *DefaultManager) MigrateKeyStoreDir(oldDir, newDir string, addresses []string) error {
	paths := []string{}

	addressesMap := map[string]struct{}{}
	for _, address := range addresses {
		addressesMap[address] = struct{}{}
	}

	checkFile := func(path string, fileInfo os.FileInfo) error {
		if fileInfo.IsDir() || filepath.Dir(path) != oldDir {
			return nil
		}

		rawKeyFile, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("invalid account key file: %v", err)
		}

		var accountKey struct {
			Address string `json:"address"`
		}
		if err := json.Unmarshal(rawKeyFile, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		address := ethtypes.HexToAddress("0x" + accountKey.Address).Hex()
		if _, ok := addressesMap[address]; ok {
			paths = append(paths, path)
		}

		return nil
	}

	err := filepath.Walk(oldDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return checkFile(path, fileInfo)
	})
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	for _, path := range paths {
		_, fileName := filepath.Split(path)
		newPath := filepath.Join(newDir, fileName)
		err := os.Rename(path, newPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *DefaultManager) ReEncryptKey(rawKey []byte, pass string, newPass string) (reEncryptedKey []byte, e error) {
	cryptoJSON, e := keystore.RawKeyToCryptoJSON(rawKey)
	if e != nil {
		return reEncryptedKey, fmt.Errorf("convert to crypto json error: %v", e)
	}

	decryptedKey, e := keystore.DecryptKey(rawKey, pass)
	if e != nil {
		return reEncryptedKey, fmt.Errorf("decryption error: %v", e)
	}

	if cryptoJSON.KDFParams["n"] == nil || cryptoJSON.KDFParams["p"] == nil {
		return reEncryptedKey, fmt.Errorf("Unable to determine `n` or `p`: %v", e)
	}
	n := int(cryptoJSON.KDFParams["n"].(float64))
	p := int(cryptoJSON.KDFParams["p"].(float64))

	gethKey := gethkeystore.Key{
		Id:              decryptedKey.ID,
		Address:         gethcommon.Address(decryptedKey.Address),
		PrivateKey:      decryptedKey.PrivateKey,
		ExtendedKey:     decryptedKey.ExtendedKey,
		SubAccountIndex: decryptedKey.SubAccountIndex,
	}

	return gethkeystore.EncryptKey(&gethKey, newPass, n, p)
}

func (m *DefaultManager) ReEncryptKeyStoreDir(keyDirPath, oldPass, newPass string) error {
	rencryptFileAtPath := func(tempKeyDirPath, path string, fileInfo os.FileInfo) error {
		if fileInfo.IsDir() {
			return nil
		}

		rawKeyFile, e := ioutil.ReadFile(path)
		if e != nil {
			return fmt.Errorf("invalid account key file: %v", e)
		}

		reEncryptedKey, e := m.ReEncryptKey(rawKeyFile, oldPass, newPass)
		if e != nil {
			return fmt.Errorf("unable to re-encrypt key file: %v, path: %s, name: %s", e, path, fileInfo.Name())
		}

		tempWritePath := filepath.Join(tempKeyDirPath, fileInfo.Name())
		e = ioutil.WriteFile(tempWritePath, reEncryptedKey, fileInfo.Mode().Perm())
		if e != nil {
			return fmt.Errorf("unable write key file: %v", e)
		}

		return nil
	}

	keyDirPath = strings.TrimSuffix(keyDirPath, "/")
	keyDirPath = strings.TrimSuffix(keyDirPath, "\\")
	keyParent, keyDirName := filepath.Split(keyDirPath)

	// backupKeyDirName used to store existing keys before final write
	backupKeyDirName := keyDirName + "-backup"
	// tempKeyDirName used to put re-encrypted keys
	tempKeyDirName := keyDirName + "-re-encrypted"
	backupKeyDirPath := filepath.Join(keyParent, backupKeyDirName)
	tempKeyDirPath := filepath.Join(keyParent, tempKeyDirName)

	// create temp key dir
	err := os.MkdirAll(tempKeyDirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("mkdirall error: %v, tempKeyDirPath: %s", err, tempKeyDirPath)
	}

	err = filepath.Walk(keyDirPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			os.RemoveAll(tempKeyDirPath)
			return fmt.Errorf("walk callback error: %v", err)
		}

		return rencryptFileAtPath(tempKeyDirPath, path, fileInfo)
	})
	if err != nil {
		os.RemoveAll(tempKeyDirPath)
		return fmt.Errorf("walk error: %v", err)
	}

	// move existing keys
	err = os.Rename(keyDirPath, backupKeyDirPath)
	if err != nil {
		os.RemoveAll(tempKeyDirPath)
		return fmt.Errorf("unable to rename keyDirPath to backupKeyDirPath: %v", err)
	}

	// move tempKeyDirPath to keyDirPath
	err = os.Rename(tempKeyDirPath, keyDirPath)
	if err != nil {
		// if this happens, then the app is probably bricked, because the keystore won't exist anymore
		// try to restore from backup
		_ = os.Rename(backupKeyDirPath, keyDirPath)
		return fmt.Errorf("unable to rename tempKeyDirPath to keyDirPath: %v", err)
	}

	// remove temp and backup folders and their contents
	err = os.RemoveAll(tempKeyDirPath)
	if err != nil {
		// the re-encryption is complete so we don't throw
		m.logger.Error("unable to delete tempKeyDirPath, manual cleanup required")
	}

	err = os.RemoveAll(backupKeyDirPath)
	if err != nil {
		// the re-encryption is complete so we don't throw
		m.logger.Error("unable to delete backupKeyDirPath, manual cleanup required")
	}

	return nil
}

func (m *DefaultManager) DeleteAccount(address ethtypes.Address) error {
	accounts := m.keystore.Accounts()
	for _, acc := range accounts {
		if acc.Address == address {
			return m.keystore.Delete(acc)
		}
	}
	return fmt.Errorf("account not found: %s", address.Hex())
}

func (m *DefaultManager) GetVerifiedWalletAccount(db *accounts.Database, address, password string) (*types.SelectedExtKey, error) {
	exists, err := db.AddressExists(ethtypes.HexToAddress(address))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("account doesn't exist")
	}

	key, err := m.VerifyAccountPassword(m.Keydir, address, password)
	if _, ok := err.(*ErrCannotLocateKeyFile); ok {
		key, err = m.generatePartialAccountKey(db, address, password)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	return &types.SelectedExtKey{
		Address:    key.Address,
		AccountKey: key,
	}, nil
}

func (m *DefaultManager) generatePartialAccountKey(db *accounts.Database, address string, password string) (*ethtypes.Key, error) {
	dbPath, err := db.GetPath(ethtypes.HexToAddress(address))
	path := "m/" + dbPath[strings.LastIndex(dbPath, "/")+1:]
	if err != nil {
		return nil, err
	}

	rootAddress, err := db.GetWalletRootAddress()
	if err != nil {
		return nil, err
	}
	info, err := m.AccountsGenerator().LoadAccount(rootAddress.Hex(), password)
	if err != nil {
		return nil, err
	}
	masterID := info.ID

	accInfosMap, err := m.AccountsGenerator().StoreDerivedAccounts(masterID, password, []string{path})
	if err != nil {
		return nil, err
	}

	_, key, err := m.AddressToDecryptedAccount(accInfosMap[path].Address, password)
	if err != nil {
		return nil, err
	}

	return key, nil
}
