package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
)

func TestCallRPC_UntrustedBindsURLToOrigin(t *testing.T) {
	state := setupTests(t)

	legit := "https://legit-dapp.test"
	require.NoError(t, persistence.UpsertDApp(state.walletDb, &persistence.DApp{
		URL:           legit,
		Name:          "Legit",
		IconURL:       "https://legit-dapp.test/icon.png",
		SharedAccount: types.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID:       1,
	}))
	require.NoError(t, persistence.InsertPermission(state.walletDb, legit, "", commands.Method_EthAccounts, nil, 1))

	// Browser Origin is evil, but JSON url claims to be legit.
	ctx := WithConnectionType(context.Background(), ConnectionTypeUntrusted)
	ctx = WithRequestOrigin(ctx, "https://evil.test")

	request := `{
		"method": "eth_accounts",
		"params": [],
		"url": "https://legit-dapp.test",
		"name": "Legit",
		"iconUrl": "https://legit-dapp.test/icon.png"
	}`
	res, err := state.api.CallRPC(ctx, request)
	require.NoError(t, err)
	// Bound to evil Origin: no permission → empty accounts (not legit's shared account).
	accounts, ok := res.([]string)
	require.True(t, ok)
	require.Empty(t, accounts)
}
