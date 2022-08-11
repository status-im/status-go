package appdatabase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/sqlite"
)

func Test_GetDBFilename(t *testing.T) {
	// Test with a temp file instance
	db, stop, err := SetupTestSQLDB("test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, stop())
	}()

	fn, err := GetDBFilename(db)
	require.NoError(t, err)
	require.True(t, len(fn) > 0)

	// Test with in memory instance
	mdb, err := InitializeDB(":memory:", "test", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, mdb.Close())
	}()

	fn, err = GetDBFilename(mdb)
	require.NoError(t, err)
	require.Equal(t, "", fn)
}
