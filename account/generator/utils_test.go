package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/extkeys"
)

func generateTestKey(t *testing.T) *extkeys.ExtendedKey {
	mnemonic := extkeys.NewMnemonic()
	mnemonicPhrase, err := mnemonic.MnemonicPhrase(extkeys.EntropyStrength128, extkeys.EnglishLanguage)
	require.NoError(t, err)

	masterExtendedKey, err := extkeys.NewMaster(mnemonic.MnemonicSeed(mnemonicPhrase, ""))
	require.NoError(t, err)

	return masterExtendedKey
}

func TestValidateKeystoreExtendedKey(t *testing.T) {
	extendedKey1 := generateTestKey(t)
	extendedKey2 := generateTestKey(t)

	// new keystore file format
	key := &types.Key{
		PrivateKey:  extendedKey1.ToECDSA(),
		ExtendedKey: extendedKey1,
	}
	assert.NoError(t, ValidateKeystoreExtendedKey(key))

	// old keystore file format where the extended key was
	// from another derivation path and not the same of the private key
	oldKey := &types.Key{
		PrivateKey:  extendedKey1.ToECDSA(),
		ExtendedKey: extendedKey2,
	}
	assert.Error(t, ValidateKeystoreExtendedKey(oldKey))

	// normal key where we don't have an extended key
	normalKey := &types.Key{
		PrivateKey:  extendedKey1.ToECDSA(),
		ExtendedKey: nil,
	}
	assert.NoError(t, ValidateKeystoreExtendedKey(normalKey))
}
