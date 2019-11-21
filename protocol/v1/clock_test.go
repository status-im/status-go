package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalcMessageClockWithoutLastObservedValue(t *testing.T) {
	result := CalcMessageClock(0, 1)
	require.Equal(t, clockBumpInMs+1, result)
}
