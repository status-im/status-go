package node

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

// Regression: Start marks the node running before startup can fail. If it does
// not roll that back, IsRunning lies and a later Stop tears down a node that
// was never fully built, dereferencing fields that were never assigned.
func TestStartFailureResetsRunning(t *testing.T) {
	config, err := params.NewNodeConfig("", walletcommon.EthereumSepolia)
	require.NoError(t, err)
	// No active networks: TokenManager creation fails, so startWithDB returns early.
	config.Networks = nil

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

	require.Error(t, n.Start(config), "expected startup to fail with no active networks")
	require.False(t, n.IsRunning(), "a failed Start must not leave the node marked running")
}
