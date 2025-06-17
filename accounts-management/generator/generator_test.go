package generator

import (
	"strings"
	"testing"

	"github.com/status-im/extkeys"

	"github.com/stretchr/testify/assert"

	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type testAccount struct {
	path       string
	privateKey string
	publicKey  string
	address    string
	keyUID     string
}

var testData = struct {
	mnemonic           string
	bip39Passphrase    string
	encriptionPassword string
	extendedMasterKey  string

	masterAccount testAccount
	childAccount0 testAccount
	childAccount1 testAccount
}{
	mnemonic:           "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
	bip39Passphrase:    "TREZOR",
	encriptionPassword: "TEST_PASSWORD",
	extendedMasterKey:  "xprv9s21ZrQH143K3h3fDYiay8mocZ3afhfULfb5GX8kCBdno77K4HiA15Tg23wpbeF1pLfs1c5SPmYHrEpTuuRhxMwvKDwqdKiGJS9XFKzUsAF",
	masterAccount: testAccount{
		path:       "m",
		privateKey: "0xcbedc75b0d6412c85c79bc13875112ef912fd1e756631b5a00330866f22ff184",
		publicKey:  "0x04f632717d78bf73e74aa8461e2e782532abae4eed5110241025afb59ebfd3d2fd55dcbc97ce588a492a152798460f89dfeacf266b3cb544bf216ec1e3e3c766e0",
		address:    "0xd7FDc6389223c747de571b22C2860B6b1CE9643a",
		keyUID:     "0x6b33d8b321bedcccd9bb3eeda0383c9ee49437fec19452e3b87f4efe36e57c70",
	},
	childAccount0: testAccount{
		path:       "m/44'/60'/0'/0/0",
		privateKey: "0x62f1d86b246c81bdd8f6c166d56896a4a5e1eddbcaebe06480e5c0bc74c28224",
		publicKey:  "0x04986dee3b8afe24cb8ccb2ac23dac3f8c43d22850d14b809b26d6b8aa5a1f47784152cd2c7d9edd0ab20392a837464b5a750b2a7f3f06e6a5756b5211b6a6ed05",
		address:    "0x9c32F71D4DB8Fb9e1A58B0a80dF79935e7256FA6",
		keyUID:     "0x06d6639c5b0fb5465d80e97efe9288b0b046223fc33b054c1083946a21f49315",
	},
	childAccount1: testAccount{
		path:       "m/44'/60'/0'/0/1",
		privateKey: "0x49ee230b1605382ac1c40079191bca937fc30e8c2fa845b7de27a96ffcc4ddbf",
		publicKey:  "0x04462e7b95dab24fe8a57ac897d9026545ec4327c9c5e4a772e5d14cc5422f94896d222a9e8880e41562c41e8290b842679d33c450bb5329caa3f078fbdf9e639d",
		address:    "0x7AF7283bd1462C3b957e8FAc28Dc19cBbF2FAdfe",
		keyUID:     "0x5bd1c31af7a92087f76fa0def8ded5277d4d69b0aa4f4a4352912f4026aa68c3",
	},
}

func createKey(t *testing.T, seedPhrase string, bip39Passphrase string) *ethtypes.Key {
	mnemonic := extkeys.NewMnemonic()
	seed := mnemonic.MnemonicSeed(seedPhrase, bip39Passphrase)

	masterKey, err := extkeys.NewMaster(seed)
	assert.NoError(t, err)

	return &ethtypes.Key{
		PrivateKey:  masterKey.ToECDSA(),
		ExtendedKey: masterKey,
	}
}

func TestGenerator_CreateAccountFromMnemonic(t *testing.T) {
	acc, err := CreateAccountFromMnemonic(testData.mnemonic, testData.bip39Passphrase)
	assert.NoError(t, err)
	assert.Equal(t, testData.extendedMasterKey, acc.extendedKey.String())

	info := acc.ToIdentifiedAccountInfo()
	assert.Equal(t, testData.masterAccount.keyUID, info.KeyUID)
	assert.Equal(t, testData.masterAccount.address, info.Address)
	assert.Equal(t, testData.masterAccount.publicKey, info.PublicKey)
	assert.Equal(t, testData.masterAccount.privateKey, info.PrivateKey)
}
func TestGenerator_CreateAccountFromPrivateKey(t *testing.T) {
	acc, err := CreateAccountFromPrivateKey(testData.masterAccount.privateKey)
	assert.NoError(t, err)

	info := acc.ToIdentifiedAccountInfo()
	assert.Equal(t, testData.masterAccount.keyUID, info.KeyUID)
	assert.Equal(t, testData.masterAccount.address, info.Address)
	assert.Equal(t, testData.masterAccount.publicKey, info.PublicKey)
	assert.Equal(t, testData.masterAccount.privateKey, info.PrivateKey)
}

func TestGenerator_CreateAccountFromKey(t *testing.T) {
	key := createKey(t, testData.mnemonic, testData.bip39Passphrase)

	acc, err := CreateAccountFromKey(key)
	assert.NoError(t, err)
	assert.Equal(t, key.ExtendedKey.String(), acc.extendedKey.String())

	info := acc.ToIdentifiedAccountInfo()
	assert.Equal(t, testData.masterAccount.keyUID, info.KeyUID)
	assert.Equal(t, testData.masterAccount.address, info.Address)
	assert.Equal(t, testData.masterAccount.publicKey, info.PublicKey)
	assert.Equal(t, testData.masterAccount.privateKey, info.PrivateKey)
}

func TestGenerator_CreateAccountsOfMnemonicLength(t *testing.T) {
	accounts, mnemonicPhrases, err := CreateAccountsOfMnemonicLength(12, 5, "")
	assert.NoError(t, err)
	assert.Equal(t, 5, len(accounts))
	assert.Equal(t, len(accounts), len(mnemonicPhrases))

	for i, acc := range accounts {
		info := acc.ToGeneratedAccountInfo(mnemonicPhrases[i])
		words := strings.Split(info.Mnemonic, " ")
		assert.Equal(t, 12, len(words))
	}
}

func TestGenerator_DeriveChildFromAccount(t *testing.T) {
	acc, err := CreateAccountFromMnemonic(testData.mnemonic, testData.bip39Passphrase)
	assert.NoError(t, err)

	child, err := DeriveChildFromAccount(acc, testData.childAccount0.path)
	assert.NoError(t, err)

	info := child.ToIdentifiedAccountInfo()

	assert.Equal(t, testData.childAccount0.keyUID, info.KeyUID)
	assert.Equal(t, testData.childAccount0.address, info.Address)
	assert.Equal(t, testData.childAccount0.publicKey, info.PublicKey)
	assert.Equal(t, testData.childAccount0.privateKey, info.PrivateKey)
}

func TestGenerator_DeriveChildrenFromAccount(t *testing.T) {
	acc, err := CreateAccountFromMnemonic(testData.mnemonic, testData.bip39Passphrase)
	assert.NoError(t, err)

	children, err := DeriveChildrenFromAccount(acc, []string{testData.masterAccount.path, testData.childAccount0.path, testData.childAccount1.path})
	assert.NoError(t, err)

	assert.Equal(t, 3, len(children))

	assert.Equal(t, testData.masterAccount.keyUID, children[testData.masterAccount.path].ToIdentifiedAccountInfo().KeyUID)
	assert.Equal(t, testData.masterAccount.address, children[testData.masterAccount.path].ToIdentifiedAccountInfo().Address)
	assert.Equal(t, testData.masterAccount.publicKey, children[testData.masterAccount.path].ToIdentifiedAccountInfo().PublicKey)
	assert.Equal(t, testData.masterAccount.privateKey, children[testData.masterAccount.path].ToIdentifiedAccountInfo().PrivateKey)

	assert.Equal(t, testData.childAccount0.keyUID, children[testData.childAccount0.path].ToIdentifiedAccountInfo().KeyUID)
	assert.Equal(t, testData.childAccount0.address, children[testData.childAccount0.path].ToIdentifiedAccountInfo().Address)
	assert.Equal(t, testData.childAccount0.publicKey, children[testData.childAccount0.path].ToIdentifiedAccountInfo().PublicKey)
	assert.Equal(t, testData.childAccount0.privateKey, children[testData.childAccount0.path].ToIdentifiedAccountInfo().PrivateKey)

	assert.Equal(t, testData.childAccount1.keyUID, children[testData.childAccount1.path].ToIdentifiedAccountInfo().KeyUID)
	assert.Equal(t, testData.childAccount1.address, children[testData.childAccount1.path].ToIdentifiedAccountInfo().Address)
	assert.Equal(t, testData.childAccount1.publicKey, children[testData.childAccount1.path].ToIdentifiedAccountInfo().PublicKey)
	assert.Equal(t, testData.childAccount1.privateKey, children[testData.childAccount1.path].ToIdentifiedAccountInfo().PrivateKey)
}
