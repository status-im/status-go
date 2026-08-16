package geth

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/status-im/extkeys"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto"
)

func TestDecryptPrivateKeySkipsExtendedKey(t *testing.T) {
	const password = "password"

	extendedKey, err := extkeys.NewMaster([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	privateKey := extendedKey.ToECDSA()
	key := &types.Key{
		ID:              uuid.New(),
		Address:         crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey:      privateKey,
		ExtendedKey:     extendedKey,
		SubAccountIndex: 7,
	}

	keyJSON, err := EncryptKey(key, password, 2, 1)
	require.NoError(t, err)

	var encryptedKey encryptedKeyJSONV3
	require.NoError(t, json.Unmarshal(keyJSON, &encryptedKey))
	encryptedKey.ExtendedKey.MAC = "invalid"
	keyJSON, err = json.Marshal(encryptedKey)
	require.NoError(t, err)

	_, err = DecryptKey(keyJSON, password)
	require.Error(t, err)

	decryptedKey, err := DecryptPrivateKey(keyJSON, password)
	require.NoError(t, err)
	require.Equal(t, key.ID, decryptedKey.ID)
	require.Equal(t, key.Address, decryptedKey.Address)
	require.Equal(t, key.PrivateKey.D, decryptedKey.PrivateKey.D)
	require.Nil(t, decryptedKey.ExtendedKey)
	require.Equal(t, key.SubAccountIndex, decryptedKey.SubAccountIndex)
}

func TestDecryptPrivateKeyRejectsWrongPassword(t *testing.T) {
	extendedKey, err := extkeys.NewMaster([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	privateKey := extendedKey.ToECDSA()
	keyJSON, err := EncryptKey(&types.Key{
		ID:          uuid.New(),
		Address:     crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey:  privateKey,
		ExtendedKey: extendedKey,
	}, "password", 2, 1)
	require.NoError(t, err)

	_, err = DecryptPrivateKey(keyJSON, "wrong password")
	require.Error(t, err)
}
