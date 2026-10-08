package assets

import (
	"crypto/sha256"
	"encoding/hex"
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

// SHA256SUMS pins the exact bytes shipped, so a payload change is a reviewed manifest change.
// Regenerate with: shasum -a 256 *.rawpayload > SHA256SUMS
func TestPayloadsMatchManifest(t *testing.T) {
	manifest, err := os.ReadFile("SHA256SUMS")
	require.NoError(t, err)

	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		fields := strings.Fields(line)
		require.Len(t, fields, 2, "malformed SHA256SUMS line %q", line)
		want[fields[1]] = fields[0]
	}

	files, err := filepath.Glob("*.rawpayload")
	require.NoError(t, err)
	require.Len(t, want, len(files), "SHA256SUMS and the shipped .rawpayload files differ")
	for _, f := range files {
		payload, ok := Payload(strings.TrimSuffix(f, ".rawpayload"))
		require.True(t, ok, f)
		sum := sha256.Sum256(payload)
		require.Equal(t, want[f], hex.EncodeToString(sum[:]), "%s does not match SHA256SUMS", f)
	}
}
