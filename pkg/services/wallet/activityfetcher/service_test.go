package activityfetcher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

type fakeManager struct {
	release chan struct{}
	err     error
	calls   atomic.Int32
}

func (m *fakeManager) IsChainSupported(chainID uint64) bool { return true }

func (m *fakeManager) FetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address, currentBlock uint64) (thirdparty.ActivityEntryContainer, error) {
	m.calls.Add(1)
	select {
	case <-ctx.Done():
		return thirdparty.ActivityEntryContainer{}, ctx.Err()
	case <-m.release:
	}
	return thirdparty.ActivityEntryContainer{}, m.err
}

func (m *fakeManager) GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address gethcommon.Address) (*gethrpc.BlockNumber, *time.Time, error) {
	return nil, nil, nil
}

func (m *fakeManager) ClearAll(ctx context.Context) error { return nil }

type fakeEthClientGetter struct {
	client ethclient.EthClientInterface
}

func (g *fakeEthClientGetter) EthClient(chainID uint64) (ethclient.EthClientInterface, error) {
	return g.client, nil
}

func newTestService(t *testing.T, manager *fakeManager) *Service {
	ctrl := gomock.NewController(t)
	client := mock_ethclient.NewMockEthClientInterface(ctrl)
	client.EXPECT().BlockNumber(gomock.Any()).Return(uint64(100), nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return &Service{
		activityFetcherManager: manager,
		ethClientGetter:        &fakeEthClientGetter{client: client},
		cancelFnMap:            make(map[fetcherID]*fetchRun),
		logger:                 zap.NewNop(),
		ctx:                    ctx,
	}
}

var testKey = FetchKey{Address: gethcommon.HexToAddress("0x1"), ChainID: 10}

func TestFetchAndWaitStartsFetch(t *testing.T) {
	manager := &fakeManager{release: make(chan struct{})}
	close(manager.release)
	s := newTestService(t, manager)

	complete := s.FetchAndWait(context.Background(), []FetchKey{testKey})
	require.True(t, complete[testKey])
	require.EqualValues(t, 1, manager.calls.Load())
}

func TestFetchAndWaitJoinsRunningFetch(t *testing.T) {
	manager := &fakeManager{release: make(chan struct{})}
	s := newTestService(t, manager)

	s.startFetchActivity(s.ctx, fetcherID{account: testKey.Address, chainID: testKey.ChainID})
	require.Eventually(t, func() bool { return manager.calls.Load() == 1 }, time.Second, time.Millisecond)

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(manager.release)
	}()
	complete := s.FetchAndWait(context.Background(), []FetchKey{testKey})
	require.True(t, complete[testKey])
	require.EqualValues(t, 1, manager.calls.Load())
}

func TestFetchAndWaitTimesOut(t *testing.T) {
	manager := &fakeManager{release: make(chan struct{})}
	s := newTestService(t, manager)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	complete := s.FetchAndWait(ctx, []FetchKey{testKey})
	require.False(t, complete[testKey])
	close(manager.release)
}

func TestFetchAndWaitReportsFailure(t *testing.T) {
	manager := &fakeManager{release: make(chan struct{}), err: errors.New("provider down")}
	close(manager.release)
	s := newTestService(t, manager)

	complete := s.FetchAndWait(context.Background(), []FetchKey{testKey})
	require.False(t, complete[testKey])
}

func TestFetchAndWaitFollowsRestartedFetch(t *testing.T) {
	manager := &fakeManager{release: make(chan struct{})}
	s := newTestService(t, manager)
	id := fetcherID{account: testKey.Address, chainID: testKey.ChainID}

	s.startFetchActivity(s.ctx, id)
	require.Eventually(t, func() bool { return manager.calls.Load() == 1 }, time.Second, time.Millisecond)

	result := make(chan bool)
	go func() {
		result <- s.FetchAndWait(context.Background(), []FetchKey{testKey})[testKey]
	}()
	time.Sleep(10 * time.Millisecond)

	// A restart cancels the first fetch; the waiter follows the new one.
	s.startFetchActivity(s.ctx, id)
	require.Eventually(t, func() bool { return manager.calls.Load() == 2 }, time.Second, time.Millisecond)
	require.NotNil(t, s.currentRun(id))
	close(manager.release)

	require.True(t, <-result)
}

func TestFetchAndWaitNotStarted(t *testing.T) {
	manager := &fakeManager{release: make(chan struct{})}
	s := newTestService(t, manager)
	s.ctx = nil

	complete := s.FetchAndWait(context.Background(), []FetchKey{testKey})
	require.False(t, complete[testKey])
	require.Zero(t, manager.calls.Load())
}
