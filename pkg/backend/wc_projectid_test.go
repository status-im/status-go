package backend

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestDefaultNodeConfig_CarriesWalletConnectProjectID(t *testing.T) {
	request := &requests.CreateAccount{
		RootDataDir:            t.TempDir(),
		WalletConnectProjectID: "87815d72a81d739d2a7ce15c2cfdefb3",
	}

	nodeConfig, err := DefaultNodeConfig("installation-id", "key-uid", request)
	require.NoError(t, err)
	require.Equal(t, request.WalletConnectProjectID, nodeConfig.WalletConnectProjectID)
}
