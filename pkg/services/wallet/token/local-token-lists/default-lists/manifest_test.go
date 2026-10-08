package defaulttokenlists

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// SHA256SUMS pins the exact bytes shipped, so a list change is a reviewed manifest change.
// The downloader regenerates it.
func TestEmbeddedListsMatchManifest(t *testing.T) {
	manifest, err := os.ReadFile("SHA256SUMS")
	require.NoError(t, err)

	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		fields := strings.Fields(line)
		require.Len(t, fields, 2, "malformed SHA256SUMS line %q", line)
		want[fields[1]] = fields[0]
	}

	files, err := filepath.Glob("*.json")
	require.NoError(t, err)
	require.Len(t, want, len(files), "SHA256SUMS and the shipped .json files differ")
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		sum := sha256.Sum256(data)
		require.Equal(t, want[f], hex.EncodeToString(sum[:]), "%s does not match SHA256SUMS", f)
	}

	shipped := map[string]bool{}
	for _, h := range want {
		shipped[h] = true
	}
	for _, l := range allLists() {
		sum := sha256.Sum256(l.JsonData)
		require.True(t, shipped[hex.EncodeToString(sum[:])], "embedded list %s is not one of the files in SHA256SUMS", l.ID)
	}
}
