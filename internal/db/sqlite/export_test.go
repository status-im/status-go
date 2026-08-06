package sqlite

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestDB(t *testing.T, path, key string, kdfIter int) {
	t.Helper()
	db, err := OpenDB(path, key, kdfIter)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE test (value TEXT); INSERT INTO test (value) VALUES ('hello')")
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func requireDBValue(t *testing.T, path, key string, kdfIter int) {
	t.Helper()
	db, err := OpenDB(path, key, kdfIter)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	var value string
	require.NoError(t, db.QueryRow("SELECT value FROM test").Scan(&value))
	require.Equal(t, "hello", value)
}

func TestExportDBWithKDFChange(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	dstPath := filepath.Join(dir, "dst.db")

	const (
		srcKey  = "old-password"
		dstKey  = "new-high-entropy-key"
		srcIter = 25600
		dstIter = ReducedKDFIterationsNumber
	)

	createTestDB(t, srcPath, srcKey, srcIter)

	started, finished := false, false
	require.NoError(t, ExportDBWithKDFChange(srcPath, srcKey, srcIter, dstPath, dstKey, dstIter,
		func() { started = true }, func() { finished = true }))
	require.True(t, started)
	require.True(t, finished)

	// The exported DB opens with the new key and the new iteration count...
	requireDBValue(t, dstPath, dstKey, dstIter)

	// ...and not with the old iteration count or the old key.
	_, err := OpenDB(dstPath, dstKey, srcIter)
	require.Error(t, err)
	_, err = OpenDB(dstPath, srcKey, dstIter)
	require.Error(t, err)

	// The source is untouched.
	requireDBValue(t, srcPath, srcKey, srcIter)
}

func TestExportDBKeepsKDFIterations(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	dstPath := filepath.Join(dir, "dst.db")

	createTestDB(t, srcPath, "old-password", ReducedKDFIterationsNumber)

	require.NoError(t, ExportDB(srcPath, "old-password", ReducedKDFIterationsNumber, dstPath, "new-password", nil, nil))
	requireDBValue(t, dstPath, "new-password", ReducedKDFIterationsNumber)
}
