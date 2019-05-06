package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxWritesOnCommit(t *testing.T) {
	storage, err := NewMemoryLevelDBStorage()
	tx := storage.NewTx()
	require.NoError(t, err)
	key := []byte{1}
	val := []byte{1, 1}
	require.NoError(t, tx.Put(key, val))
	result, err := storage.Get(key)
	require.Error(t, err)
	require.Nil(t, result)
	require.NoError(t, tx.Commit())
	result, err = storage.Get(key)
	require.NoError(t, err)
	require.Equal(t, val, result)
}
