package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Every shipped payload file must be registered, otherwise it is silently left out of the binary's bootstrap.
func TestEveryPayloadFileIsRegistered(t *testing.T) {
	files, err := filepath.Glob("*.rawpayload")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	ids := make([]string, 0, len(files))
	for _, f := range files {
		id := strings.TrimSuffix(f, ".rawpayload")
		ids = append(ids, id)

		want, err := os.ReadFile(f)
		require.NoError(t, err)
		got, ok := Payload(id)
		require.True(t, ok, "%s is not registered in embedded.go", f)
		require.Equal(t, want, got, "%s", f)
	}
	require.ElementsMatch(t, ids, IDs())
}
