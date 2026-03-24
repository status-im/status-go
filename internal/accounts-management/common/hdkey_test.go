package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The UR:CRYPTO-HDKEY below was exported from a Keycard shell for path m/44'/60'/0'
// from seed phrase "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about".
const testURCryptoHDKey = "UR:CRYPTO-HDKEY/OSAOWKAXHDCLAOWDVEROKOPDINHSEEROISYALKSAYKCTJSHEDPRNUYJYFGROVAWEWFTYGHCEGLRPKGAAHDCXTPLFJSLUKNFWLAISAXWYPALBJYLSWZAMCXHSCYUYLOZTMWFNLDLGSKPYPTGSDECFAMTAADDYOTADLNCSDWYKCSFNYKAEYKAOCYJKSKTNBKAXAXAYCYTEDMFEAYASJNGRIHKKIAHSJPIECXGUISIHJZJZBKJOHSIAIAJLKPJTJYDMJKJYHSJTIEHSJPIEMWLAGLZM"

// testXPub is the BIP32 xpub reconstructed from the testURCryptoHDKey, path m/44'/60'/0'
const testXPub = "xpub6DCoCpSuQZB2jawqnGMEPS63ePKWkwWPH4TU45Q7LPXWuNd8TMtVxRrgjtEshuqpK3mdhaWHPFsBngh5GFZaM6si3yZdUsT8ddYM3PwnATt"

func TestURCryptoHDKeyToXPub(t *testing.T) {
	t.Run("converts UR:CRYPTO-HDKEY to xpub", func(t *testing.T) {
		xpub, err := URCryptoHDKeyToXPub(testURCryptoHDKey)
		require.NoError(t, err)
		assert.Equal(t, testXPub, xpub)
	})

	t.Run("handles lowercase UR prefix", func(t *testing.T) {
		lower := "ur:crypto-hdkey/" + testURCryptoHDKey[len("UR:CRYPTO-HDKEY/"):]
		xpub, err := URCryptoHDKeyToXPub(lower)
		require.NoError(t, err)
		assert.Equal(t, testXPub, xpub)
	})

	t.Run("returns error for wrong UR type", func(t *testing.T) {
		_, err := URCryptoHDKeyToXPub("UR:CRYPTO-SEED/abc123")
		assert.Error(t, err)
	})

	t.Run("returns error for invalid bytewords", func(t *testing.T) {
		_, err := URCryptoHDKeyToXPub("UR:CRYPTO-HDKEY/!!invalid!!")
		assert.Error(t, err)
	})

	t.Run("returns error for checksum mismatch", func(t *testing.T) {
		// Corrupt the last character
		corrupted := testURCryptoHDKey[:len(testURCryptoHDKey)-1] + "a"
		_, err := URCryptoHDKeyToXPub(corrupted)
		assert.Error(t, err)
	})
}
