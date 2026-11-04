package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueries(t *testing.T) {
	db := NewMockSqliteDB()
	queries, err := NewQueries("test_queries", db)
	require.NoError(t, err)

	query := queries.Delete()
	require.NotEmpty(t, query)

	query = queries.Exists()
	require.NotEmpty(t, query)

	query = queries.Get()
	require.NotEmpty(t, query)

	query = queries.Put()
	require.NotEmpty(t, query)

	query = queries.Query()
	require.NotEmpty(t, query)

	query = queries.Prefix()
	require.NotEmpty(t, query)

	query = queries.Limit()
	require.NotEmpty(t, query)

	query = queries.Offset()
	require.NotEmpty(t, query)

	query = queries.GetSize()
	require.NotEmpty(t, query)
}

func TestCreateTable(t *testing.T) {
	db := NewMockSqliteDB()

	err := CreateTable(db, "test_create_table")
	require.NoError(t, err)
}
