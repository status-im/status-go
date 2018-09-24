package crypto

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	key = "0000000000000000000000000000000000000000000000000000000000000000"
)

var expectedPlaintext = []byte("test")

func TestSymmetricEncryption(t *testing.T) {
	key, err := hex.DecodeString(key)
	assert.Nil(t, err, "Key should be generated without errors")

	cyphertext1, err := EncryptSymmetric(key, expectedPlaintext)
	assert.Nil(t, err, "Cyphertext should be generated without errors")

	cyphertext2, err := EncryptSymmetric(key, expectedPlaintext)
	assert.Nil(t, err, "Cyphertext should be generated without errors")

	assert.Equalf(
		t,
		32,
		len(cyphertext1),
		"Cyphertext with the correct lenght should be generated")

	assert.NotEqualf(
		t,
		cyphertext1,
		cyphertext2,
		"Same plaintext should not be encrypted in the same way")

	plaintext, err := DecryptSymmetric(key, cyphertext1)
	assert.Nil(t, err, "Cyphertext should be decrypted without errors")

	assert.Equalf(
		t,
		expectedPlaintext,
		plaintext,
		"Cypther text should be decrypted successfully")
}
