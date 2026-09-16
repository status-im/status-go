//go:build !test_fast_kdf

package envelope

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProductionScryptParams pins the production KDF cost with literal values: scrypt allocates a
// 128*N*r scratch buffer per derivation, so these parameters fix the 64 MiB memory budget and the
// password-hardening strength. Changing any of them must be a deliberate decision that updates
// this test.
func TestProductionScryptParams(t *testing.T) {
	dir := t.TempDir()

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))

	content, err := os.ReadFile(Path(dir, testKeyUID))
	require.NoError(t, err)
	var file wrappedKeyFile
	require.NoError(t, json.Unmarshal(content, &file))

	require.Equal(t, "scrypt", file.Crypto.KDF)
	require.EqualValues(t, 65536, file.Crypto.KDFParams["n"])
	require.EqualValues(t, 1, file.Crypto.KDFParams["p"])
	require.EqualValues(t, 8, file.Crypto.KDFParams["r"])
	require.EqualValues(t, 32, file.Crypto.KDFParams["dklen"])
}
