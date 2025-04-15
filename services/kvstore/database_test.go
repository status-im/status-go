package kvstore

import (
	"database/sql"
	"testing"

	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"
)

type DbInitializer struct {
}

func (a DbInitializer) Initialize(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	db, err := sqlite.OpenDB(path, password, kdfIterationsNumber)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kv_store (
		key TEXT PRIMARY KEY,
		value BLOB
	);`)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func setupTestDB(t *testing.T) (*Database, func()) {
	db, err := helpers.SetupTestMemorySQLDB(DbInitializer{})
	require.NoError(t, err)
	return NewDB(db), func() { require.NoError(t, db.Close()) }
}

func TestSetGetDelete(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.Set(TestDemoKey, []byte("value"))
	require.NoError(t, err)

	value, err := db.Get(TestDemoKey)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), value)

	err = db.Set(TestDemoKey, []byte("another-value"))
	require.NoError(t, err)

	value, err = db.Get(TestDemoKey)
	require.NoError(t, err)
	require.Equal(t, []byte("another-value"), value)

	err = db.Delete(TestDemoKey)
	require.NoError(t, err)

	value, err = db.Get(TestDemoKey)
	require.NoError(t, err)
	require.Nil(t, value)
}

func TestSetBoolGetBool(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	value, err := db.GetBool(TestDemoKey)
	require.NoError(t, err)
	require.False(t, value)

	err = db.SetBool(TestDemoKey, true)
	require.NoError(t, err)

	value, err = db.GetBool(TestDemoKey)
	require.NoError(t, err)
	require.True(t, value)

	err = db.SetBool(TestDemoKey, false)
	require.NoError(t, err)

	value, err = db.GetBool(TestDemoKey)
	require.NoError(t, err)
	require.False(t, value)
}

func TestSetNotUsedKey(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.Set("not_used_key", []byte("value"))
	require.Error(t, err)
	require.Equal(t, "key not in usage", err.Error())
}

func TestDropDeprecatedKeys(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	err := db.Set(TestDemoKey, []byte("value"))
	require.NoError(t, err)

	err = db.DropDeprecatedKeys([]string{TestDemoKey})
	require.NoError(t, err)

	value, err := db.Get(TestDemoKey)
	require.NoError(t, err)
	require.Nil(t, value)
}
