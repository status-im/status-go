package wallet_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet"
)

// TestAPI_HashMessageEIP191
func TestAPI_HashMessageEIP191(t *testing.T) {
	api := &wallet.API{}

	res := api.HashMessageEIP191(context.Background(), []byte("test"))
	require.Equal(t, "0x4a5c5d454721bbbb25540c3317521e71c373ae36458f960d2ad46ef088110e95", res.String())
}

func TestAPI_IsChecksumValidForAddress(t *testing.T) {
	api := &wallet.API{}

	res, err := api.IsChecksumValidForAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)
	require.False(t, res)

	res, err = api.IsChecksumValidForAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa")
	require.NoError(t, err)
	require.True(t, res)
}
