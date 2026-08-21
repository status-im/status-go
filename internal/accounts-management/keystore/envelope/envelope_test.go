package envelope

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testKeyUID = "0x1122334455667788990011223344556677889900112233445566778899001122"
	testKEK    = "0x20756e6465727374616e64207468652063757272656e74206265686176696f72"
	otherKEK   = "0x6f74686572206b656b206f74686572206b656b206f74686572206b656b202121"
)

func TestGenerate(t *testing.T) {
	dek1, err := Generate()
	require.NoError(t, err)
	require.Len(t, dek1, 2*DekLength)

	dek2, err := Generate()
	require.NoError(t, err)
	require.NotEqual(t, dek1, dek2)
}

func TestReadDBKdfIterations(t *testing.T) {
	dir := t.TempDir()

	_, err := ReadDBKdfIterations(dir, testKeyUID)
	require.Error(t, err)

	dek, err := Generate()
	require.NoError(t, err)
	require.NoError(t, Write(dir, testKeyUID, dek, testKEK, 3200))

	iterations, err := ReadDBKdfIterations(dir, testKeyUID)
	require.NoError(t, err)
	require.Equal(t, 3200, iterations)
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
