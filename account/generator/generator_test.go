package generator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/crypto"
)

var testAccount = struct {
	mnemonic           string
	bip39Passphrase    string
	encriptionPassword string
	extendedMasterKey  string
	bip44Key0          string
	bip44PubKey0       string
	bip44Address0      string
	bip44Address1      string
}{
	mnemonic:           "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
	bip39Passphrase:    "TREZOR",
	encriptionPassword: "TEST_PASSWORD",
	extendedMasterKey:  "xprv9s21ZrQH143K3h3fDYiay8mocZ3afhfULfb5GX8kCBdno77K4HiA15Tg23wpbeF1pLfs1c5SPmYHrEpTuuRhxMwvKDwqdKiGJS9XFKzUsAF",
	bip44Key0:          "0x62f1d86b246c81bdd8f6c166d56896a4a5e1eddbcaebe06480e5c0bc74c28224",
	bip44PubKey0:       "0x04986dee3b8afe24cb8ccb2ac23dac3f8c43d22850d14b809b26d6b8aa5a1f47784152cd2c7d9edd0ab20392a837464b5a750b2a7f3f06e6a5756b5211b6a6ed05",
	bip44Address0:      "0x9c32F71D4DB8Fb9e1A58B0a80dF79935e7256FA6",
	bip44Address1:      "0x7AF7283bd1462C3b957e8FAc28Dc19cBbF2FAdfe",
}

const (
	path0 = "m/44'/60'/0'/0/0"
	path1 = "m/44'/60'/0'/0/1"
)

func TestGenerator_Generate(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	accountsInfo, err := g.Generate(12, 5, "")
	assert.NoError(t, err)
	assert.Equal(t, 5, len(g.accounts))

	for _, info := range accountsInfo {
		words := strings.Split(info.Mnemonic, " ")
		assert.Equal(t, 12, len(words))
	}
}

func TestGenerator_CreateAccountFromPrivateKey(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	info, err := g.CreateAccountFromPrivateKey(testAccount.bip44Key0)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(g.accounts))
	assert.Equal(t, 66, len(info.KeyUID))
}

func TestGenerator_ImportPrivateKey(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	info, err := g.ImportPrivateKey(testAccount.bip44Key0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(g.accounts))

	assert.Equal(t, testAccount.bip44PubKey0, info.PublicKey)
	assert.Equal(t, testAccount.bip44Address0, info.Address)
}

func TestGenerator_CreateAccountFromMnemonicAndDeriveAccountsForPaths(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	info, err := g.CreateAccountFromMnemonicAndDeriveAccountsForPaths(testAccount.mnemonic, testAccount.bip39Passphrase, []string{path0, path1})

	assert.NoError(t, err)
	assert.Equal(t, 0, len(g.accounts))
	assert.Equal(t, 66, len(info.KeyUID))

	assert.Equal(t, testAccount.bip44Address0, info.Derived[path0].Address)
	assert.Equal(t, testAccount.bip44Address1, info.Derived[path1].Address)
}

func TestGenerator_ImportMnemonic(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	info, err := g.ImportMnemonic(testAccount.mnemonic, testAccount.bip39Passphrase)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(g.accounts))

	key := g.accounts[info.ID]
	assert.Equal(t, testAccount.extendedMasterKey, key.extendedKey.String())
}

func TestGenerator_DeriveAddresses(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	info, err := g.ImportMnemonic(testAccount.mnemonic, testAccount.bip39Passphrase)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(g.accounts))

	addresses, err := g.DeriveAddresses(info.ID, []string{path0, path1})
	assert.NoError(t, err)

	assert.Equal(t, testAccount.bip44Address0, addresses[path0].Address)
	assert.Equal(t, testAccount.bip44Address1, addresses[path1].Address)
}

func TestGenerator_DeriveAddresses_FromImportedPrivateKey(t *testing.T) {
	g := New(nil)
	assert.Equal(t, 0, len(g.accounts))

	key, err := crypto.GenerateKey()
	assert.NoError(t, err)
	hex := fmt.Sprintf("%#x", crypto.FromECDSA(key))
	info, err := g.ImportPrivateKey(hex)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(g.accounts))

	// normal imported accounts cannot derive child accounts,
	// but only the address/pubblic key of the current key.
	paths := []string{"", "m"}
	for _, path := range paths {
		addresses, err := g.DeriveAddresses(info.ID, []string{path})
		assert.NoError(t, err)

		expectedAddress := crypto.PubkeyToAddress(key.PublicKey).Hex()
		assert.Equal(t, expectedAddress, addresses[path].Address)
	}

	// cannot derive other child keys from a normal key
	_, err = g.DeriveAddresses(info.ID, []string{"m/0/1/2"})
	assert.Equal(t, ErrAccountCannotDeriveChildKeys, err)
}
