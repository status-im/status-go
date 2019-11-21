package alias

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenerate(t *testing.T) {
	var seed uint64 = 42

	name := generate(seed)
	require.NotNil(t, name)
	require.Equal(t, "Hard Tame Brownbutterfly", name)
}

func TestGenerateFromPublicKeyString(t *testing.T) {
	pk := "0x04eedbaafd6adf4a9233a13e7b1c3c14461fffeba2e9054b8d456ce5f6ebeafadcbf3dce3716253fbc391277fa5a086b60b283daf61fb5b1f26895f456c2f31ae3"

	name, err := GenerateFromPublicKeyString(pk)
	require.NoError(t, err)
	require.NotNil(t, name)
	require.Equal(t, "Darkorange Blue Bubblefish", name)
}
