package appdatabase

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetDBFilename(t *testing.T) {
	// Test with a temp file instance
	db, stop, err := SetupTestSQLDB("test")
	defer func() {
		require.NoError(t, stop())
	}()

	fn, err := GetDBFilename(db)
	require.NoError(t, err)
	require.True(t, len(fn) > 0)

	// Test with in memory instance
	mdb, err := InitializeDB(":memory:", "test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, mdb.Close())
	}()

	fn, err = GetDBFilename(mdb)
	require.NoError(t, err)
	require.Equal(t, "", fn)
}

