package walletconnect

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestDecryptType0Envelope(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plaintext := []byte(`{"jsonrpc":"2.0","method":"wc_sessionPropose"}`)

	encrypted, err := EncryptType0Envelope(symKeyHex, plaintext)
	require.NoError(t, err)

	decrypted, err := DecryptType0Envelope(symKeyHex, encrypted)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestDecryptType0Envelope_InvalidSymKey(t *testing.T) {
	encrypted := base64.StdEncoding.EncodeToString([]byte{0x00, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})

	_, err := DecryptType0Envelope("invalid", encrypted)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode symkey")
}

func TestDecryptType0Envelope_ShortSymKey(t *testing.T) {
	encrypted := base64.StdEncoding.EncodeToString([]byte{0x00, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})

	_, err := DecryptType0Envelope("0123", encrypted)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid symkey length")
}

func TestDecryptType0Envelope_InvalidBase64(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	_, err := DecryptType0Envelope(symKeyHex, "not-valid-base64!!!")
	require.Error(t, err)
	require.Contains(t, err.Error(), "base64 decode")
}

func TestDecryptType0Envelope_TooShort(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	envelope := []byte{0x00, 1, 2}
	encrypted := base64.StdEncoding.EncodeToString(envelope)

	_, err := DecryptType0Envelope(symKeyHex, encrypted)
	require.Error(t, err)
	require.Contains(t, err.Error(), "envelope too short")
}

func TestDecryptType0Envelope_WrongType(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	envelope := make([]byte, 50)
	envelope[0] = 0x99
	encrypted := base64.StdEncoding.EncodeToString(envelope)

	_, err := DecryptType0Envelope(symKeyHex, encrypted)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported envelope type")
}

func TestDecryptType0Envelope_CorruptedData(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	envelope := make([]byte, 50)
	envelope[0] = 0x00
	encrypted := base64.StdEncoding.EncodeToString(envelope)

	_, err := DecryptType0Envelope(symKeyHex, encrypted)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decrypt")
}

func TestEncryptType0Envelope(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plaintext := []byte("test message")

	encrypted, err := EncryptType0Envelope(symKeyHex, plaintext)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	envelope, err := base64.StdEncoding.DecodeString(encrypted)
	require.NoError(t, err)
	require.Equal(t, byte(0x00), envelope[0])
	require.GreaterOrEqual(t, len(envelope), 1+12+len(plaintext)+chacha20poly1305.Overhead)
}

func TestEncryptType0Envelope_InvalidSymKey(t *testing.T) {
	_, err := EncryptType0Envelope("invalid", []byte("test"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode symkey")
}

func TestEncryptType0Envelope_ShortSymKey(t *testing.T) {
	_, err := EncryptType0Envelope("0123", []byte("test"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid symkey length")
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte(nil)},
		{"short", []byte("x")},
		{"json", []byte(`{"id":123,"method":"test"}`)},
		{"long", []byte(string(make([]byte, 10000)))},
		{"unicode", []byte("Hello 世界 🌍")},
	}

	symKeyHex := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptType0Envelope(symKeyHex, tt.plaintext)
			require.NoError(t, err)

			decrypted, err := DecryptType0Envelope(symKeyHex, encrypted)
			require.NoError(t, err)
			require.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptType0Envelope_DifferentIVs(t *testing.T) {
	symKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plaintext := []byte("test message")

	enc1, err := EncryptType0Envelope(symKeyHex, plaintext)
	require.NoError(t, err)

	enc2, err := EncryptType0Envelope(symKeyHex, plaintext)
	require.NoError(t, err)

	require.NotEqual(t, enc1, enc2, "IVs should be random")
}

func TestDecryptType0Envelope_WrongKey(t *testing.T) {
	symKey1 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	symKey2 := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	plaintext := []byte("secret message")

	encrypted, err := EncryptType0Envelope(symKey1, plaintext)
	require.NoError(t, err)

	_, err = DecryptType0Envelope(symKey2, encrypted)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decrypt")
}
