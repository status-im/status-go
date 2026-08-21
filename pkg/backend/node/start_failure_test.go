package node

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/rpc/network/testutil"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/params"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

// Regression: Start marks the node running before startup can fail. If it does
// not roll that back, IsRunning lies and a later Stop tears down a node that
// was never fully built, dereferencing fields that were never assigned.
func TestStartFailureResetsRunning(t *testing.T) {
	config, err := params.NewNodeConfig("", walletcommon.EthereumSepolia)
	require.NoError(t, err)
	// No active networks: token.NewTokenManager fails ("chains are not
	// provided"), so startWithDB returns after the RPC client, downloader and
	// media server were already built.
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

	// A failed Start must also unwind what startWithDB built before failing,
	// not just roll the flag back.
	require.Nil(t, n.rpcClient, "a failed Start must stop and release the RPC client")
	require.Nil(t, n.downloader, "a failed Start must stop and release the downloader")
	require.Nil(t, n.config, "a failed Start must not leave a config on a node that never started")
	require.ErrorIs(t, n.Stop(), ErrNoRunningNode, "Stop after a failed Start must report no running node")
}

// startFailingTimeSource fails at Start after initServices has constructed
// services and registerService has published their APIs on rpcServer.
type startFailingTimeSource struct {
	err error
}

func (s *startFailingTimeSource) Now() time.Time              { return time.Now() }
func (s *startFailingTimeSource) Start(context.Context) error { return s.err }
func (s *startFailingTimeSource) Stop()                       {}

var errTimeSourceStart = errors.New("time source start failed")

func startableTestNode(t *testing.T) (*StatusNode, *params.NodeConfig) {
	t.Helper()

	config, err := params.NewNodeConfig("", walletcommon.EthereumSepolia)
	require.NoError(t, err)
	config.Networks = testutil.MinimalActiveNetworks()

	n := New(nil, nil, testutils.MustCreateTestLogger())
	appDB, walletDB, stop, err := setupTestDBs()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := stop(); err != nil {
			n.logger.Error("stopping db", zap.Error(err))
		}
	})
	n.appDB = appDB
	n.walletDB = walletDB
	return n, config
}

func failStartAfterServicesRegistered(t *testing.T, n *StatusNode, config *params.NodeConfig) {
	t.Helper()
	n.timeSourceSrvc = &startFailingTimeSource{err: errTimeSourceStart}
	err := n.Start(config)
	require.ErrorIs(t, err, errTimeSourceStart)
	require.False(t, n.IsRunning(), "a failed Start must not leave the node marked running")
}

// Login retry reuses the same StatusNode: StartNodeWithAccount calls StopNode
// on error, and StopNode is a no-op when !IsRunning. The second Start
// currently "succeeds" while keeping eth (and other services) that still
// point at the RPC client teardown just stopped.
func TestStartFailureAllowsRetry(t *testing.T) {
	n, config := startableTestNode(t)
	failStartAfterServicesRegistered(t, n, config)

	staleEth := n.ethSrvc
	require.NotNil(t, staleEth, "initServices must have constructed eth before the injected failure")

	n.timeSourceSrvc = timesource.LocalService()
	require.NoError(t, n.Start(config), "a failed Start must leave the node startable again")
	require.True(t, n.IsRunning())
	require.NotSame(t, staleEth, n.ethSrvc, "retry must reconstruct services that captured the torn-down RPC client")
	require.NoError(t, n.Stop())
}

// Services constructed during the failed attempt keep the torn-down RPC
// client. teardown nils some fields but not eth/gif/chat/updates/sharedUrls/
// linkPreview, so a retry reuses those instances.
func TestStartFailureDropsServicesBoundToRPCClient(t *testing.T) {
	n, config := startableTestNode(t)
	failStartAfterServicesRegistered(t, n, config)

	require.Nil(t, n.ethSrvc, "teardown must drop eth, it holds the stopped RPC client")
	require.Nil(t, n.gifSrvc, "teardown must drop gif")
	require.Nil(t, n.chatSrvc, "teardown must drop chat")
	require.Nil(t, n.updatesSrvc, "teardown must drop updates")
	require.Nil(t, n.sharedUrlsSrvc, "teardown must drop sharedUrls")
	require.Nil(t, n.linkPreviewSrvc, "teardown must drop linkPreview")
}
