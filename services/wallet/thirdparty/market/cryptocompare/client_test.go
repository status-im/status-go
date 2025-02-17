package cryptocompare

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDs(t *testing.T) {
	stdClient := NewClient()
	require.Equal(t, baseID, stdClient.ID())

	clientWithParams := NewClientWithParams(Params{
		ID: "testID",
	})
	require.Equal(t, "testID", clientWithParams.ID())
}
