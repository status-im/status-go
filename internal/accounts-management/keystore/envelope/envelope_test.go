package envelope

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	geth "github.com/status-im/status-go/internal/accounts-management/keystore/internal/geth"
)

const (
	testKeyUID = "0x1122334455667788990011223344556677889900112233445566778899001122"
	testKEK    = "0x20756e6465727374616e64207468652063757272656e74206265686176696f72"
	otherKEK   = "0x6f74686572206b656b206f74686572206b656b206f74686572206b656b202121"
)

func TestGenerate(t *testing.T) {
	dek1, err := Generate()
	require.NoError(t, err)
	require.Len(t, dek1, 2*dekLength)

	dek2, err := Generate()
	require.NoError(t, err)
	require.NotEqual(t, dek1, dek2)
}

func TestWriteUnwrapRoundTrip(t *testing.T) {
	dir := t.TempDir()

	dek, err := Generate()
	require.NoError(t, err)

	require.False(t, Exists(dir, testKeyUID))
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))
	require.True(t, Exists(dir, testKeyUID))

	unwrapped, kdfIterations, err := Unwrap(dir, testKeyUID, testKEK)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
	require.Equal(t, 3200, kdfIterations)
}

func TestUnwrapWrongKEK(t *testing.T) {
	dir := t.TempDir()

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))

	_, _, err = Unwrap(dir, testKeyUID, otherKEK)
	require.ErrorIs(t, err, ErrInvalidKEK)
}

func TestUnwrapMissingFile(t *testing.T) {
	_, _, err := Unwrap(t.TempDir(), testKeyUID, testKEK)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRewrap(t *testing.T) {
	dir := t.TempDir()

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))

	// Wrong old KEK must not touch the file.
	require.ErrorIs(t, Rewrap(dir, testKeyUID, otherKEK, "irrelevant"), ErrInvalidKEK)
	unwrapped, _, err := Unwrap(dir, testKeyUID, testKEK)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)

	require.NoError(t, Rewrap(dir, testKeyUID, testKEK, otherKEK))

	_, _, err = Unwrap(dir, testKeyUID, testKEK)
	require.ErrorIs(t, err, ErrInvalidKEK)

	unwrapped, kdfIterations, err := Unwrap(dir, testKeyUID, otherKEK)
	require.NoError(t, err)
	require.Equal(t, dek, unwrapped)
	require.Equal(t, 3200, kdfIterations)
}

func TestWriteReplacesAtomically(t *testing.T) {
	dir := t.TempDir()

	dek1, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek1, testKEK, 3200))

	dek2, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek2, otherKEK, 3200))

	unwrapped, _, err := Unwrap(dir, testKeyUID, otherKEK)
	require.NoError(t, err)
	require.Equal(t, dek2, unwrapped)

	// No temp file left behind.
	matches, err := filepath.Glob(Path(dir, testKeyUID) + ".*.tmp")
	require.NoError(t, err)
	require.Empty(t, matches)
}

func TestUnwrapMalformedFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(Path(dir, testKeyUID), []byte("not json"), 0600))

	_, _, err := Unwrap(dir, testKeyUID, testKEK)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrInvalidKEK)
}

func TestUnwrapUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))

	content, err := os.ReadFile(Path(dir, testKeyUID))
	require.NoError(t, err)
	var file map[string]interface{}
	require.NoError(t, json.Unmarshal(content, &file))
	file["version"] = 999
	content, err = json.Marshal(file)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(Path(dir, testKeyUID), content, 0600))

	_, _, err = Unwrap(dir, testKeyUID, testKEK)
	require.ErrorContains(t, err, "unsupported wrapped-DEK file version")
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()

	// Removing a non-existent file is not an error.
	require.NoError(t, Remove(dir, testKeyUID))

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))
	require.NoError(t, os.WriteFile(Path(dir, testKeyUID)+".123456.tmp", []byte("leftover"), 0600))

	require.NoError(t, Remove(dir, testKeyUID))
	require.False(t, Exists(dir, testKeyUID))
	matches, err := filepath.Glob(Path(dir, testKeyUID) + ".*.tmp")
	require.NoError(t, err)
	require.Empty(t, matches)
}

func TestPathNaming(t *testing.T) {
	require.Equal(t, filepath.Join("/data", testKeyUID+"-profile.kek"), Path("/data", testKeyUID))
}

// TestWriteRecordsScryptParams pins the KDF cost recorded in newly written envelopes: this KDF is
// the profile's sole offline brute-force barrier, and its N caps the 128*N*r scrypt memory spike.
func TestWriteRecordsScryptParams(t *testing.T) {
	dir := t.TempDir()

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))

	content, err := os.ReadFile(Path(dir, testKeyUID))
	require.NoError(t, err)
	var file wrappedKeyFile
	require.NoError(t, json.Unmarshal(content, &file))

	require.Equal(t, "scrypt", file.Crypto.KDF)
	require.EqualValues(t, scryptN, file.Crypto.KDFParams["n"])
	require.EqualValues(t, scryptP, file.Crypto.KDFParams["p"])
}

// TestUnwrapForeignScryptParams verifies that a file written under a different KDF cost than the
// compiled constants still unwraps: the parameters live inside the file, so changing the constants
// never invalidates existing envelopes.
func TestUnwrapForeignScryptParams(t *testing.T) {
	dir := t.TempDir()

	dekHex, err := Generate()
	require.NoError(t, err)
	dek, err := hex.DecodeString(dekHex)
	require.NoError(t, err)

	const foreignN, foreignP = 4096, 6
	cryptoJSON, err := geth.EncryptDataV3(dek, []byte(testKEK), foreignN, foreignP)
	require.NoError(t, err)
	content, err := json.Marshal(wrappedKeyFile{
		Version:         fileVersion,
		KeyUID:          testKeyUID,
		DBKdfIterations: 3200,
		Crypto:          cryptoJSON,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(Path(dir, testKeyUID), content, 0600))

	unwrapped, kdfIterations, err := Unwrap(dir, testKeyUID, testKEK)
	require.NoError(t, err)
	require.Equal(t, dekHex, unwrapped)
	require.Equal(t, 3200, kdfIterations)

	// A password change (Rewrap) of such a file keeps the DEK, moves it to the new KEK and
	// re-writes the file with the currently compiled parameters.
	require.NoError(t, Rewrap(dir, testKeyUID, testKEK, otherKEK))

	unwrapped, kdfIterations, err = Unwrap(dir, testKeyUID, otherKEK)
	require.NoError(t, err)
	require.Equal(t, dekHex, unwrapped)
	require.Equal(t, 3200, kdfIterations)

	_, _, err = Unwrap(dir, testKeyUID, testKEK)
	require.ErrorIs(t, err, ErrInvalidKEK)

	content, err = os.ReadFile(Path(dir, testKeyUID))
	require.NoError(t, err)
	var rewritten wrappedKeyFile
	require.NoError(t, json.Unmarshal(content, &rewritten))
	require.EqualValues(t, scryptN, rewritten.Crypto.KDFParams["n"])
	require.EqualValues(t, scryptP, rewritten.Crypto.KDFParams["p"])
}
