//go:build tkl

package node

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/networks/testutil"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func TestStatusNodeStartWithNimCatalogue(t *testing.T) { testStatusNodeStart(t, true) }

func TestInvalidNimConfigCustomClosesCatalogue(t *testing.T) {
	n := New(nil, nil, testutils.MustCreateTestLogger())
	app, wallet, cleanup, err := setupTestDBs()
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()
	n.appDB, n.walletDB = app, wallet
	n.config = &params.NodeConfig{Networks: testutil.MinimalActiveNetworks(), WalletConfig: params.WalletConfig{TokenListsUseNim: true, CustomTokens: []*tokentypes.Token{{Token: &types.Token{ChainID: 1, Decimals: 256, Symbol: "BAD"}}}}}
	require.NoError(t, n.setupRPCClient())
	defer n.rpcClient.Stop()
	require.Error(t, n.createTokenManager())
	require.NotNil(t, n.tokenManager)
	defer n.tokenManager.Stop()
	require.ErrorContains(t, n.tokenManager.Start(context.Background()), "stopped")
}
