package kvstore

import (
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "kvstore-tests")
	require.NoError(t, err)
	return NewDB(db), func() { require.NoError(t, cleanup()) }
}

func TestSetGetDelete(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.Set("key", []byte("value"))
	require.NoError(t, err)

	value, err := db.Get("key")
	require.NoError(t, err)
	require.Equal(t, []byte("value"), value)

	err = db.Set("key", []byte("another-value"))
	require.NoError(t, err)

	value, err = db.Get("key")
	require.NoError(t, err)
	require.Equal(t, []byte("another-value"), value)

	err = db.Delete("key")
	require.NoError(t, err)

	value, err = db.Get("key")
	require.NoError(t, err)
	require.Nil(t, value)
}

func TestSetBoolGetBool(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	value, err := db.GetBool("key")
	require.NoError(t, err)
	require.False(t, value)

	err = db.SetBool("key", true)
	require.NoError(t, err)

	value, err = db.GetBool("key")
	require.NoError(t, err)
	require.True(t, value)

	err = db.SetBool("key", false)
	require.NoError(t, err)

	value, err = db.GetBool("key")
	require.NoError(t, err)
	require.False(t, value)
}

func TestDropDeprecatedKeys(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.Set("key", []byte("value"))
	require.NoError(t, err)

	err = db.Set("deprecated-key", []byte("value"))
	require.NoError(t, err)

	err = db.DropDeprecatedKeys([]string{"deprecated-key"})
	require.NoError(t, err)

	value, err := db.Get("key")
	require.NoError(t, err)
	require.Equal(t, []byte("value"), value)

	value, err = db.Get("deprecated-key")
	require.NoError(t, err)
	require.Nil(t, value)
}
