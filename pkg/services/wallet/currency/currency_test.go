package currency

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateTokenDisplayDecimals(t *testing.T) {
	require.EqualValues(t, 0, calculateTokenDisplayDecimals(0.001))
	require.EqualValues(t, 0, calculateTokenDisplayDecimals(0.01))
	require.EqualValues(t, 0, calculateTokenDisplayDecimals(0.015))
	require.EqualValues(t, 1, calculateTokenDisplayDecimals(0.1))
	require.EqualValues(t, 1, calculateTokenDisplayDecimals(0.3))
	require.EqualValues(t, 2, calculateTokenDisplayDecimals(1))
	require.EqualValues(t, 2, calculateTokenDisplayDecimals(5))
	require.EqualValues(t, 3, calculateTokenDisplayDecimals(10))
	require.EqualValues(t, 3, calculateTokenDisplayDecimals(80))
	require.EqualValues(t, 4, calculateTokenDisplayDecimals(100))
	require.EqualValues(t, 4, calculateTokenDisplayDecimals(365))
	require.EqualValues(t, 5, calculateTokenDisplayDecimals(1000))
	require.EqualValues(t, 5, calculateTokenDisplayDecimals(6548))
	require.EqualValues(t, 6, calculateTokenDisplayDecimals(10000))
	require.EqualValues(t, 6, calculateTokenDisplayDecimals(54623))
	require.EqualValues(t, 7, calculateTokenDisplayDecimals(100000))
	require.EqualValues(t, 7, calculateTokenDisplayDecimals(986315))
}
