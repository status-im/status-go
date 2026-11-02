package node

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/networks/testutil"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

// Regression: Start marks the node running before startup can fail. If it does
// not roll that back, IsRunning lies and a later Stop tears down a node that
// was never fully built, dereferencing fields that were never assigned.
func TestStartFailureResetsRunning(t *testing.T) {
	config, err := params.NewNodeConfig("", walletcommon.EthereumSepolia)
	require.NoError(t, err)
	config.Networks = testutil.MinimalActiveNetworks()
	// An invalid persisted custom token fails TokenManager creation, so
	// startWithDB returns after the RPC client, downloader and media server
	// were already built.
	config.WalletConfig.CustomTokens = []*tokentypes.Token{{Token: &types.Token{ChainID: 1, Decimals: 256}}}

	n := New(nil, nil, testutils.MustCreateTestLogger())

	appDB, walletDB, stop, err := setupTestDBs()
	require.NoError(t, err)
	defer func() {
		if err := stop(); err != nil {
			n.logger.Error("stopping db", zap.Error(err))
		}
	}()
	n.appDB = appDB
	n.walletDB = walletDB
	defer func() { require.NoError(t, n.StopMediaServer()) }()

	require.Error(t, n.Start(config), "expected startup to fail with an invalid custom token")
	require.False(t, n.IsRunning(), "a failed Start must not leave the node marked running")

	// A failed Start must also unwind what startWithDB built before failing,
	// not just roll the flag back.
	require.Nil(t, n.rpcClient, "a failed Start must stop and release the RPC client")
	require.Nil(t, n.downloader, "a failed Start must stop and release the downloader")
	require.Nil(t, n.config, "a failed Start must not leave a config on a node that never started")
	require.ErrorIs(t, n.Stop(), ErrNoRunningNode, "Stop after a failed Start must report no running node")
}
