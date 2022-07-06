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

	"github.com/google/uuid"

	gethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/extkeys"
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

type ErrCannotLocateKeyFile struct {
	Msg string
}

func (e ErrCannotLocateKeyFile) Error() string {
	return e.Msg
}

var zeroAddress = types.Address{}

// Manager represents account manager interface.
type Manager struct {
	mu       sync.RWMutex
	keystore types.KeyStore

	accountsGenerator *generator.Generator
	onboarding        *Onboarding

	selectedChatAccount *SelectedExtKey // account that was processed during the last call to SelectAccount()
	mainAccountAddress  types.Address
	watchAddresses      []types.Address
}

// GetKeystore is only used in tests
func (m *Manager) GetKeystore() types.KeyStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keystore
}

// AccountsGenerator returns accountsGenerator.
func (m *Manager) AccountsGenerator() *generator.Generator {
	return m.accountsGenerator
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (m *Manager) CreateAccount(password string) (generator.GeneratedAccountInfo, Info, string, error) {
	var mkInfo generator.GeneratedAccountInfo
	info := Info{}

	// generate mnemonic phrase
	mn := extkeys.NewMnemonic()
	mnemonic, err := mn.MnemonicPhrase(extkeys.EntropyStrength128, extkeys.EnglishLanguage)
	if err != nil {
		return mkInfo, info, "", fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	// Generate extended master key (see BIP32)
	// We call extkeys.NewMaster with a seed generated with the 12 mnemonic words
	// but without using the optional password as an extra entropy as described in BIP39.
	// Future ideas/iterations in Status can add an an advanced options
	// for expert users, to be able to add a passphrase to the generation of the seed.
	extKey, err := extkeys.NewMaster(mn.MnemonicSeed(mnemonic, ""))
	if err != nil {
		return mkInfo, info, "", fmt.Errorf("can not create master extended key: %v", err)
	}

	acc := generator.NewAccount(nil, extKey)
	mkInfo = acc.ToGeneratedAccountInfo("", mnemonic)

	// import created key into account keystore
	info.WalletAddress, info.WalletPubKey, err = m.importExtendedKey(extkeys.KeyPurposeWallet, extKey, password)
	if err != nil {
		return mkInfo, info, "", err
	}

	info.ChatAddress = info.WalletAddress
	info.ChatPubKey = info.WalletPubKey

	return mkInfo, info, mnemonic, nil
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
func (m *Manager) VerifyAccountPassword(keyStoreDir, address, password string) (*types.Key, error) {
	var err error
	var foundKeyFile []byte

	addressObj := types.BytesToAddress(types.FromHex(address))
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
		if types.HexToAddress("0x"+accountKey.Address).Hex() == addressObj.Hex() {
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

func (m *Manager) SetAccountAddresses(main types.Address, secondary ...types.Address) {
	m.watchAddresses = []types.Address{main}
	m.watchAddresses = append(m.watchAddresses, secondary...)
	m.mainAccountAddress = main
}

// SetChatAccount initializes selectedChatAccount with privKey
func (m *Manager) SetChatAccount(privKey *ecdsa.PrivateKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	address := crypto.PubkeyToAddress(privKey.PublicKey)
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	key := &types.Key{
		ID:         id,
		Address:    address,
		PrivateKey: privKey,
	}

	m.selectedChatAccount = &SelectedExtKey{
		Address:    address,
		AccountKey: key,
	}
	return nil
}

// MainAccountAddress returns main account address set during login
func (m *Manager) MainAccountAddress() (types.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.mainAccountAddress == zeroAddress {
		return zeroAddress, ErrNoAccountSelected
	}

	return m.mainAccountAddress, nil
}

// WatchAddresses returns currently selected watch addresses.
func (m *Manager) WatchAddresses() []types.Address {
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
func (m *Manager) ImportAccount(privateKey *ecdsa.PrivateKey, password string) (types.Address, error) {
	if m.keystore == nil {
		return types.Address{}, ErrAccountKeyStoreMissing
	}

	account, err := m.keystore.ImportECDSA(privateKey, password)

	return account.Address, err
}

// ImportSingleExtendedKey imports an extended key setting it in both the PrivateKey and ExtendedKey fields
// of the Key struct.
// ImportExtendedKey is used in older version of Status where PrivateKey is set to be the BIP44 key at index 0,
// and ExtendedKey is the extended key of the BIP44 key at index 1.
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

	pubKey = types.EncodeHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

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
	pubKey = types.EncodeHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}

// Accounts returns list of addresses for selected account, including
// subaccounts.
func (m *Manager) Accounts() ([]types.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	addresses := make([]types.Address, 0)
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
func (m *Manager) AddressToDecryptedAccount(address, password string) (types.Account, *types.Key, error) {
	if m.keystore == nil {
		return types.Account{}, nil, ErrAccountKeyStoreMissing
	}

	account, err := ParseAccountString(address)
	if err != nil {
		return types.Account{}, nil, ErrAddressToAccountMappingFailure
	}

	account, key, err := m.keystore.AccountDecryptedKey(account, password)
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

func (m *Manager) MigrateKeyStoreDir(oldDir, newDir string, addresses []string) error {
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

		address := types.HexToAddress("0x" + accountKey.Address).Hex()
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

func (m *Manager) ReEncryptKey(rawKey []byte, pass string, newPass string) (reEncryptedKey []byte, e error) {
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

func (m *Manager) ReEncryptKeyStoreDir(keyDirPath, oldPass, newPass string) error {
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
		log.Error("unable to delete tempKeyDirPath, manual cleanup required")
	}

	err = os.RemoveAll(backupKeyDirPath)
	if err != nil {
		// the re-encryption is complete so we don't throw
		log.Error("unable to delete backupKeyDirPath, manual cleanup required")
	}

	return nil
}

func (m *Manager) DeleteAccount(keyDirPath string, address types.Address) error {
	var err error
	var foundKeyFile string
	err = filepath.Walk(keyDirPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
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

		if types.HexToAddress("0x"+accountKey.Address).Hex() == address.Hex() {
			foundKeyFile = path
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	if len(foundKeyFile) == 0 {
		return ErrCannotLocateKeyFile{fmt.Sprintf("cannot locate account for address: %s", address.Hex())}
	}
	return os.Remove(foundKeyFile)
}
