package sqlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlaceholders(t *testing.T) {
	require.Equal(t, "", Placeholders(0))
	require.Equal(t, "", Placeholders(-1))
	require.Equal(t, "?", Placeholders(1))
	require.Equal(t, "?, ?, ?", Placeholders(3))
}

func TestIn(t *testing.T) {
	require.Equal(t, "DELETE FROM t WHERE id IN (?, ?)", In("DELETE FROM t WHERE id IN (%s)", 2))
	require.Equal(t,
		"SELECT 1 FROM t WHERE a IN (?) AND b IN (?, ?, ?)",
		In("SELECT 1 FROM t WHERE a IN (%s) AND b IN (%s)", 1, 3))
}
