package alias

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	var seed uint64 = 42

	name := generate(seed)
	require.NotNil(t, name)
	require.Equal(t, "Hard Tame Brownbutterfly", name)
}

func TestGenerateFromPublicKeyString(t *testing.T) {
	tests := []struct {
		name          string
		publicKey     string
		alias         string
		errorExpected bool
	}{
		{
			name:          "valid public key - start with 0x",
			publicKey:     "0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3",
			alias:         "Darkorange Blue Bubblefish",
			errorExpected: false,
		},
		{
			name:          "valid public key - without 0x",
			publicKey:     "04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3",
			alias:         "Darkorange Blue Bubblefish",
			errorExpected: false,
		},
		{
			name:          "invalid public key",
			publicKey:     "0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b",
			alias:         "",
			errorExpected: true,
		},
		{
			name:          "empty public key",
			publicKey:     "",
			alias:         "",
			errorExpected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := GenerateFromPublicKeyString(tt.publicKey)
			if tt.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.alias, name)
		})
	}
}
