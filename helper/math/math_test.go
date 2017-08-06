package math

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRound(t *testing.T) {
	// Arrange.

	// Act.
	rounded := Round(0.8400029999999999, 6)

	// Assert.
	require.Equal(t, 0.840003, rounded)
}
