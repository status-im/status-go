package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCapabilitiesCommand(t *testing.T) {
	cmd := NewGetCapabilitiesCommand()
	_, err := cmd.Execute(context.Background(), RPCRequest{})
	require.ErrorIs(t, err, ErrRequestMissingDAppData)

	out, err := cmd.Execute(context.Background(), RPCRequest{
		URL: testDAppData.URL, Name: testDAppData.Name, IconURL: testDAppData.IconURL,
	})
	require.NoError(t, err)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	require.Empty(t, m)
}
