package crypto

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	const content1 = "045a8cae84d8d139e887bb927d2b98cee481afae3770e0ee45f2dc19c6545e45921bc6a55ea92b705e45dfbbe47182c7b1d64a080a220d2781577163923d7cbb4b045a8cae84d8d139e887bb927d2b98cee481afae3770e0ee45f2dc19c6545e45921bc6a55ea92b705e45dfbbe47182c7b1d64a080a220d2781577163923d7cbb4b04ca82dd41fa592bf46ecf7e2eddae61013fc95a565b59c49f37f06b1b591ed3bd24e143495f2d1e241e151ab3572ac108d577be349d4b88d3d5a50c481ab35441"
	const content2 = "045a8cae84d8d139e887bb927d2b98cee481afae3770e0ee45f2dc19c6545e45921bc6a55ea92b705e45dfbbe47182c7b1d64a080a220d2781577163923d7cbb4b045a8cae84d8d139e887bb927d2b98cee481afae3770e0ee45f2dc19c6545e45921bc6a55ea92b705e45dfbbe47182c7b1d64a080a220d2781577163923d7cbb4b04ca82dd41fa592bf46ecf7e2eddae61013fc95a565b59c49f37f06b1b591ed3bd24e143495f2d1e241e151ab3572ac108d577be349d4b88d3d5a50c481ab35440"

	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	signature1, err := Sign(content1, key1)
	require.NoError(t, err)
	fmt.Println(signature1)

	signature2, err := Sign(content2, key2)
	require.NoError(t, err)

	key1String := hex.EncodeToString(crypto.FromECDSAPub(&key1.PublicKey))
	key2String := hex.EncodeToString(crypto.FromECDSAPub(&key2.PublicKey))

	pair1 := [3]string{content1, signature1, key1String}
	pair2 := [3]string{content2, signature2, key2String}

	signaturePairs := [][3]string{pair1, pair2}

	err = VerifySignatures(signaturePairs)
	require.NoError(t, err)

	// Test wrong content
	pair3 := [3]string{content1, signature2, key2String}

	signaturePairs = [][3]string{pair1, pair2, pair3}

	err = VerifySignatures(signaturePairs)
	require.Error(t, err)

	// Test wrong signature
	pair3 = [3]string{content1, signature2, key1String}

	signaturePairs = [][3]string{pair1, pair2, pair3}

	err = VerifySignatures(signaturePairs)
	require.Error(t, err)

	// Test wrong pubkey
	pair3 = [3]string{content1, signature1, key2String}

	signaturePairs = [][3]string{pair1, pair2, pair3}

	err = VerifySignatures(signaturePairs)
	require.Error(t, err)
}

func TestSymmetricEncryption(t *testing.T) {
	const rawKey = "0000000000000000000000000000000000000000000000000000000000000000"
	expectedPlaintext := []byte("test")
	key, err := hex.DecodeString(rawKey)
	require.Nil(t, err, "Key should be generated without errors")

	cyphertext1, err := EncryptSymmetric(key, expectedPlaintext)
	require.Nil(t, err, "Cyphertext should be generated without errors")

	cyphertext2, err := EncryptSymmetric(key, expectedPlaintext)
	require.Nil(t, err, "Cyphertext should be generated without errors")

	require.Equalf(
		t,
		32,
		len(cyphertext1),
		"Cyphertext with the correct lenght should be generated")

	require.NotEqualf(
		t,
		cyphertext1,
		cyphertext2,
		"Same plaintext should not be encrypted in the same way")

	plaintext, err := DecryptSymmetric(key, cyphertext1)
	require.Nil(t, err, "Cyphertext should be decrypted without errors")

	require.Equalf(
		t,
		expectedPlaintext,
		plaintext,
		"Cypther text should be decrypted successfully")
}
