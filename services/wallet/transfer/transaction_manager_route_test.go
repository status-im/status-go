package transfer

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	crypto2 "github.com/status-im/status-go/internal/crypto"
	statusErrors "github.com/status-im/status-go/internal/errors"
	"github.com/status-im/status-go/services/wallet/requests"
)

const testSignatureKey = "0xc8e7a34af766c4ba9dc9b3d49939806fbf41fa01250c5a26afa5659e87b2020b"

// rsBytes builds recognisable, non-uniform R and S values so a one-byte shift in the
// assembled signature is visible.
func rsBytes() (rBytes, sBytes []byte) {
	rBytes = make([]byte, 32)
	sBytes = make([]byte, 32)
	for i := 0; i < 32; i++ {
		rBytes[i] = byte(i + 1)
		sBytes[i] = byte(0x81 + i)
	}
	return rBytes, sBytes
}

func TestSignatureFor_Layout(t *testing.T) {
	rBytes, sBytes := rsBytes()

	tests := []struct {
		name   string
		offset recoveryOffset
		v      string
		wantV  byte
	}{
		{"transaction, odd parity", recoveryOffsetRaw, "01", 1},
		{"transaction, even parity", recoveryOffsetRaw, "00", 0},
		{"permit, odd parity", recoveryOffsetEIP712, "01", 28},
		{"permit, even parity", recoveryOffsetEIP712, "00", 27},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signatures := map[string]requests.SignatureDetails{
				testSignatureKey: {R: hex.EncodeToString(rBytes), S: hex.EncodeToString(sBytes), V: tt.v},
			}

			signature, err := signatureFor(testSignatureKey, signatures, tt.offset)

			require.NoError(t, err)
			require.Len(t, signature, crypto2.SignatureLength)
			require.Equal(t, rBytes, signature[:32], "R occupies the first 32 bytes")
			require.Equal(t, sBytes, signature[32:64], "S occupies the second 32 bytes")
			require.Equal(t, tt.wantV, signature[64])
		})
	}
}

func TestSignatureFor_RejectsMalformedHex(t *testing.T) {
	rBytes, sBytes := rsBytes()

	tests := []struct {
		name string
		r    string
		s    string
	}{
		{"R not hex", hex.EncodeToString(rBytes[:31]) + "zz", hex.EncodeToString(sBytes)},
		{"S not hex", hex.EncodeToString(rBytes), hex.EncodeToString(sBytes[:31]) + "zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signatures := map[string]requests.SignatureDetails{
				testSignatureKey: {R: tt.r, S: tt.s, V: "01"},
			}

			signature, err := signatureFor(testSignatureKey, signatures, recoveryOffsetRaw)

			require.Error(t, err)
			require.Nil(t, signature)
		})
	}
}

func TestSignatureFor_RejectsMissingAndInvalidDetails(t *testing.T) {
	rBytes, sBytes := rsBytes()

	t.Run("no signature for the key", func(t *testing.T) {
		signature, err := signatureFor(testSignatureKey, map[string]requests.SignatureDetails{}, recoveryOffsetRaw)
		require.Error(t, err)
		var errResponse *statusErrors.ErrorResponse
		require.ErrorAs(t, err, &errResponse)
		require.Equal(t, ErrMissingSignatureForTx.Code, errResponse.Code)
		require.Contains(t, errResponse.Details, testSignatureKey)
		require.Nil(t, signature)
	})

	t.Run("wrong component lengths", func(t *testing.T) {
		signatures := map[string]requests.SignatureDetails{
			testSignatureKey: {R: hex.EncodeToString(rBytes[:31]), S: hex.EncodeToString(sBytes), V: "01"},
		}
		signature, err := signatureFor(testSignatureKey, signatures, recoveryOffsetRaw)
		require.ErrorIs(t, err, requests.ErrInvalidSignatureDetails)
		require.Nil(t, signature)
	})
}
