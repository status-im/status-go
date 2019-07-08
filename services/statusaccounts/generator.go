package statusaccounts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	staccount "github.com/status-im/status-go/account"
	"github.com/status-im/status-go/extkeys"
)

var (
	errAccountNotFoundByID          = errors.New("account not found")
	errAccountCannotDeriveChildKeys = errors.New("selected account cannot derive child keys")
	errAccountManagerNotSet         = errors.New("account manager not set")
)

type generator struct {
	am       *staccount.Manager
	accounts map[string]*account
}

func newGenerator() *generator {
	return &generator{
		accounts: make(map[string]*account),
	}
}

func (g *generator) setAccountManager(am *staccount.Manager) {
	g.am = am
}

func (g *generator) generate(mnemonicPhraseLength int, n int, bip39Passphrase string) ([]CreatedAccountInfo, error) {
	entropyStrength, err := staccount.MnemonicPhraseLengthToEntropyStrenght(mnemonicPhraseLength)
	if err != nil {
		return nil, err
	}

	infos := make([]CreatedAccountInfo, 0)

	for i := 0; i < n; i++ {
		mnemonic := extkeys.NewMnemonic()
		mnemonicPhrase, err := mnemonic.MnemonicPhrase(entropyStrength, extkeys.EnglishLanguage)
		if err != nil {
			return nil, fmt.Errorf("can not create mnemonic seed: %v", err)
		}

		info, err := g.importMnemonic(mnemonicPhrase, bip39Passphrase)
		if err != nil {
			return nil, err
		}

		infos = append(infos, info)
	}

	return infos, err
}

func (g *generator) importPrivateKey(privateKeyHex string) (IdentifiedAccountInfo, error) {
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

func (g *generator) importJSONKey(json string, password string) (IdentifiedAccountInfo, error) {
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

func (g *generator) importMnemonic(mnemonicPhrase string, bip39Passphrase string) (CreatedAccountInfo, error) {
	mnemonic := extkeys.NewMnemonic()
	masterExtendedKey, err := extkeys.NewMaster(mnemonic.MnemonicSeed(mnemonicPhrase, bip39Passphrase))
	if err != nil {
		return CreatedAccountInfo{}, fmt.Errorf("can not create master extended key: %v", err)
	}

	acc := &account{
		privateKey:  masterExtendedKey.ToECDSA(),
		extendedKey: masterExtendedKey,
	}

	id := g.addAccount(acc)

	return acc.toCreatedAccountInfo(id, mnemonicPhrase), nil
}

func (g *generator) deriveAddresses(accountID string, pathStrings []string) (map[string]AccountInfo, error) {
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

func (g *generator) storeAccount(accountID string, password string) (AccountInfo, error) {
	if g.am == nil {
		return AccountInfo{}, errAccountManagerNotSet
	}

	acc, err := g.findAccount(accountID)
	if err != nil {
		return AccountInfo{}, err
	}

	return g.store(acc, password)
}

func (g *generator) storeDerivedAccounts(accountID string, password string, pathStrings []string) (map[string]AccountInfo, error) {
	if g.am == nil {
		return nil, errAccountManagerNotSet
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

func (g *generator) loadAccount(address string, password string) (IdentifiedAccountInfo, error) {
	if g.am == nil {
		return IdentifiedAccountInfo{}, errAccountManagerNotSet
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

func (g *generator) deriveChildAccounts(acc *account, pathStrings []string) (map[string]*account, error) {
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

func (g *generator) deriveChildAccount(acc *account, pathString string) (*account, error) {
	_, path, err := decodePath(pathString)
	if err != nil {
		return nil, err
	}

	if acc.extendedKey.IsZeroed() {
		return nil, errAccountCannotDeriveChildKeys
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

func (g *generator) store(acc *account, password string) (AccountInfo, error) {
	if acc.extendedKey != nil {
		if _, _, err := g.am.ImportSingleExtendedKey(acc.extendedKey, password); err != nil {
			return AccountInfo{}, err
		}
	} else {
		if _, err := g.am.ImportNormalAccount(acc.privateKey, password); err != nil {
			return AccountInfo{}, err
		}
	}

	g.Reset()

	return acc.toAccountInfo(), nil
}

func (g *generator) addAccount(acc *account) string {
	id := uuid.NewRandom().String()
	g.accounts[id] = acc

	return id
}

// Reset resets the accounts map removing all the accounts from memory.
func (g *generator) Reset() {
	g.accounts = make(map[string]*account)
}

func (g *generator) findAccount(accountID string) (*account, error) {
	acc, ok := g.accounts[accountID]
	if !ok {
		return nil, errAccountNotFoundByID
	}

	return acc, nil
}
