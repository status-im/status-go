package generator

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/extkeys"
)

var (
	// ErrAccountNotFoundByID is returned when the selected account doesn't exist in memory.
	ErrAccountNotFoundByID = errors.New("account not found")
	// ErrAccountCannotDeriveChildKeys is returned when trying to derive child accounts from a normal key.
	ErrAccountCannotDeriveChildKeys = errors.New("selected account cannot derive child keys")
	// ErrAccountManagerNotSet is returned when the account mananger instance is not set.
	ErrAccountManagerNotSet = errors.New("account manager not set")
)

type AccountManager interface {
	AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error)
	ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error)
	ImportAccount(privateKey *ecdsa.PrivateKey, password string) (common.Address, error)
}

type Generator struct {
	am       AccountManager
	accounts map[string]*account
	sync.Mutex
}

func New(am AccountManager) *Generator {
	return &Generator{
		am:       am,
		accounts: make(map[string]*account),
	}
}

func (g *Generator) Generate(mnemonicPhraseLength int, n int, bip39Passphrase string) ([]GeneratedAccountInfo, error) {
	entropyStrength, err := MnemonicPhraseLengthToEntropyStrength(mnemonicPhraseLength)
	if err != nil {
		return nil, err
	}

	infos := make([]GeneratedAccountInfo, 0)

	for i := 0; i < n; i++ {
		mnemonic := extkeys.NewMnemonic()
		mnemonicPhrase, err := mnemonic.MnemonicPhrase(entropyStrength, extkeys.EnglishLanguage)
		if err != nil {
			return nil, fmt.Errorf("can not create mnemonic seed: %v", err)
		}

		info, err := g.ImportMnemonic(mnemonicPhrase, bip39Passphrase)
		if err != nil {
			return nil, err
		}

		infos = append(infos, info)
	}

	return infos, err
}

func (g *Generator) ImportPrivateKey(privateKeyHex string) (IdentifiedAccountInfo, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return IdentifiedAccountInfo{}, err
	}

	acc := &account{
		privateKey: privateKey,
	}

	id := g.addAccount(acc)

	return acc.toIdentifiedAccountInfo(id), nil
}

func (g *Generator) ImportJSONKey(json string, password string) (IdentifiedAccountInfo, error) {
	key, err := keystore.DecryptKey([]byte(json), password)
	if err != nil {
		return IdentifiedAccountInfo{}, err
	}

	acc := &account{
		privateKey: key.PrivateKey,
	}

	id := g.addAccount(acc)

	return acc.toIdentifiedAccountInfo(id), nil
}

func (g *Generator) ImportMnemonic(mnemonicPhrase string, bip39Passphrase string) (GeneratedAccountInfo, error) {
	mnemonic := extkeys.NewMnemonic()
	masterExtendedKey, err := extkeys.NewMaster(mnemonic.MnemonicSeed(mnemonicPhrase, bip39Passphrase))
	if err != nil {
		return GeneratedAccountInfo{}, fmt.Errorf("can not create master extended key: %v", err)
	}

	acc := &account{
		privateKey:  masterExtendedKey.ToECDSA(),
		extendedKey: masterExtendedKey,
	}

	id := g.addAccount(acc)

	return acc.toGeneratedAccountInfo(id, mnemonicPhrase), nil
}

func (g *Generator) GenerateAndDeriveAddresses(mnemonicPhraseLength int, n int, bip39Passphrase string, pathStrings []string) ([]GeneratedAndDerivedAccountInfo, error) {
	masterAccounts, err := g.Generate(mnemonicPhraseLength, n, bip39Passphrase)
	if err != nil {
		return nil, err
	}

	accs := make([]GeneratedAndDerivedAccountInfo, n)

	for i := 0; i < len(masterAccounts); i++ {
		acc := masterAccounts[i]
		derived, err := g.DeriveAddresses(acc.ID, pathStrings)
		if err != nil {
			return nil, err
		}

		accs[i] = acc.toGeneratedAndDerived(derived)
	}

	return accs, nil
}

func (g *Generator) DeriveAddresses(accountID string, pathStrings []string) (map[string]AccountInfo, error) {
	acc, err := g.findAccount(accountID)
	if err != nil {
		return nil, err
	}

	pathAccounts, err := g.deriveChildAccounts(acc, pathStrings)
	if err != nil {
		return nil, err
	}

	pathAccountsInfo := make(map[string]AccountInfo)

	for pathString, childAccount := range pathAccounts {
		pathAccountsInfo[pathString] = childAccount.toAccountInfo()
	}

	return pathAccountsInfo, nil
}

func (g *Generator) StoreAccount(accountID string, password string) (AccountInfo, error) {
	if g.am == nil {
		return AccountInfo{}, ErrAccountManagerNotSet
	}

	acc, err := g.findAccount(accountID)
	if err != nil {
		return AccountInfo{}, err
	}

	return g.store(acc, password)
}

func (g *Generator) StoreDerivedAccounts(accountID string, password string, pathStrings []string) (map[string]AccountInfo, error) {
	if g.am == nil {
		return nil, ErrAccountManagerNotSet
	}

	acc, err := g.findAccount(accountID)
	if err != nil {
		return nil, err
	}

	pathAccounts, err := g.deriveChildAccounts(acc, pathStrings)
	if err != nil {
		return nil, err
	}

	pathAccountsInfo := make(map[string]AccountInfo)

	for pathString, childAccount := range pathAccounts {
		info, err := g.store(childAccount, password)
		if err != nil {
			return nil, err
		}

		pathAccountsInfo[pathString] = info
	}

	return pathAccountsInfo, nil
}

func (g *Generator) LoadAccount(address string, password string) (IdentifiedAccountInfo, error) {
	if g.am == nil {
		return IdentifiedAccountInfo{}, ErrAccountManagerNotSet
	}

	_, key, err := g.am.AddressToDecryptedAccount(address, password)
	if err != nil {
		return IdentifiedAccountInfo{}, err
	}

	if err := ValidateKeystoreExtendedKey(key); err != nil {
		return IdentifiedAccountInfo{}, err
	}

	acc := &account{
		privateKey:  key.PrivateKey,
		extendedKey: key.ExtendedKey,
	}

	id := g.addAccount(acc)

	return acc.toIdentifiedAccountInfo(id), nil
}

func (g *Generator) deriveChildAccounts(acc *account, pathStrings []string) (map[string]*account, error) {
	pathAccounts := make(map[string]*account)

	for _, pathString := range pathStrings {
		childAccount, err := g.deriveChildAccount(acc, pathString)
		if err != nil {
			return pathAccounts, err
		}

		pathAccounts[pathString] = childAccount
	}

	return pathAccounts, nil
}

func (g *Generator) deriveChildAccount(acc *account, pathString string) (*account, error) {
	_, path, err := decodePath(pathString)
	if err != nil {
		return nil, err
	}

	if acc.extendedKey.IsZeroed() && len(path) == 0 {
		return acc, nil
	}

	if acc.extendedKey.IsZeroed() {
		return nil, ErrAccountCannotDeriveChildKeys
	}

	childExtendedKey, err := acc.extendedKey.Derive(path)
	if err != nil {
		return nil, err
	}

	return &account{
		privateKey:  childExtendedKey.ToECDSA(),
		extendedKey: childExtendedKey,
	}, nil
}

func (g *Generator) store(acc *account, password string) (AccountInfo, error) {
	if acc.extendedKey != nil {
		if _, _, err := g.am.ImportSingleExtendedKey(acc.extendedKey, password); err != nil {
			return AccountInfo{}, err
		}
	} else {
		if _, err := g.am.ImportAccount(acc.privateKey, password); err != nil {
			return AccountInfo{}, err
		}
	}

	g.Reset()

	return acc.toAccountInfo(), nil
}

func (g *Generator) addAccount(acc *account) string {
	g.Lock()
	defer g.Unlock()

	id := uuid.NewRandom().String()
	g.accounts[id] = acc

	return id
}

// Reset resets the accounts map removing all the accounts from memory.
func (g *Generator) Reset() {
	g.Lock()
	defer g.Unlock()

	g.accounts = make(map[string]*account)
}

func (g *Generator) findAccount(accountID string) (*account, error) {
	g.Lock()
	defer g.Unlock()

	acc, ok := g.accounts[accountID]
	if !ok {
		return nil, ErrAccountNotFoundByID
	}

	return acc, nil
}
