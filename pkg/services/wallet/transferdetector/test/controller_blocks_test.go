package transferdetector_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/eventfilter"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"
	mock_transferdetector "github.com/status-im/status-go/pkg/services/wallet/transferdetector/mock"
)

// blockRange is a range of blocks of chain 1, both ends included.
type blockRange struct{ from, to uint64 }

// blocksHarness is a detector over one account on chain 1 whose balances were
// read at the blocks the test sets. The node is ahead of the estimate of the
// latest block, and filtering fails when the test says so.
type blocksHarness struct {
	controller *transferdetector.Controller

	mu          sync.Mutex
	first, last uint64
	read        bool
	failing     bool
	logs        []eventlog.Event
	filtered    []blockRange
}

func newBlocksHarness(t *testing.T) *blocksHarness {
	ctrl := gomock.NewController(t)
	fetcher := mock_transferdetector.NewMockFilterProvider(ctrl)
	accountsProvider := mock_transferdetector.NewMockAccountsProvider(ctrl)
	networksProvider := mock_transferdetector.NewMockNetworksProvider(ctrl)
	lastBlockProvider := mock_transferdetector.NewMockLastBlockProvider(ctrl)
	h := &blocksHarness{}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{{0x11}}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{{ChainID: 1}}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	lastBlockProvider.EXPECT().GetEstimatedLatestBlockNumber(gomock.Any(), uint64(1)).DoAndReturn(func(context.Context, uint64) (uint64, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.last + 10, nil
	}).AnyTimes()
	lastBlockProvider.EXPECT().GetReadBlocks(uint64(1)).DoAndReturn(func(uint64) (uint64, uint64, bool) {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.first, h.last, h.read
	}).AnyTimes()
	fetcher.EXPECT().FilterTransfers(gomock.Any(), uint64(1), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint64, config eventfilter.TransferQueryConfig) ([]eventlog.Event, error) {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.filtered = append(h.filtered, blockRange{config.FromBlock.Uint64(), config.ToBlock.Uint64()})
			if h.failing {
				return nil, errors.New("range too large")
			}
			logs := h.logs
			h.logs = nil
			return logs, nil
		}).AnyTimes()

	h.controller = transferdetector.NewController(transferdetector.DefaultControllerConfig(),
		fetcher, accountsProvider, pubsub.NewPublisher(), networksProvider, lastBlockProvider, zap.NewNop())
	return h
}

// readAt tells that balances were read at block, as the balance controller
// does with every result.
func (h *blocksHarness) readAt(block uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.read {
		h.first = block
	}
	h.read = true
	h.last = block
}

// mine makes the next look find logs.
func (h *blocksHarness) mine(logs ...eventlog.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.logs = append(h.logs, logs...)
}

func (h *blocksHarness) setFailing(failing bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failing = failing
}

// scan runs the detector and waits for its debounce.
func (h *blocksHarness) scan() {
	h.controller.TriggerFetch()
	h.waitDebounce()
}

func (h *blocksHarness) waitDebounce() {
	time.Sleep(11 * time.Second)
	synctest.Wait()
}

func (h *blocksHarness) takeFiltered() []blockRange {
	h.mu.Lock()
	defer h.mu.Unlock()
	filtered := h.filtered
	h.filtered = nil
	return filtered
}

// start starts the detector and waits for its first look.
func (h *blocksHarness) start() {
	h.controller.Start()
	h.waitDebounce()
}

// covered records the ranges detection reports as looked at and as gaps.
type covered struct {
	mu     sync.Mutex
	looked []blockRange
	gaps   []blockRange
}

func (h *blocksHarness) watch(t *testing.T) *covered {
	c := &covered{}
	finished, unsubFinished := pubsub.Subscribe[transferdetector.EventTransferDetectionFinished](h.controller.GetPublisher(), 1000)
	failed, unsubFailed := pubsub.Subscribe[transferdetector.EventTransferDetectionError](h.controller.GetPublisher(), 1000)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case ev := <-finished:
				c.mu.Lock()
				c.looked = append(c.looked, blockRange{ev.FromBlock, ev.ToBlock})
				c.mu.Unlock()
			case ev := <-failed:
				c.mu.Lock()
				c.gaps = append(c.gaps, blockRange{ev.FromBlock, ev.ToBlock})
				c.mu.Unlock()
			}
		}
	}()
	t.Cleanup(func() {
		close(done)
		unsubFinished()
		unsubFailed()
	})
	return c
}

func (c *covered) take() (looked []blockRange, gaps []blockRange) {
	synctest.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	looked, gaps = c.looked, c.gaps
	c.looked, c.gaps = nil, nil
	return looked, gaps
}

func TestController_NeverLooksPastTheBlockBalancesWereReadAt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		h.readAt(90)
		h.readAt(100)
		h.start()
		defer h.controller.Stop()
		require.Equal(t, []blockRange{{91, 100}}, h.takeFiltered())

		h.readAt(120)
		h.scan()

		require.Equal(t, []blockRange{{101, 120}}, h.takeFiltered())
	})
}

func TestController_LooksAtNothingBeforeBalancesAreRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		h.start()
		defer h.controller.Stop()

		h.scan()
		require.Empty(t, h.takeFiltered())

		h.readAt(100)
		h.scan()
		h.readAt(110)
		h.scan()

		require.Equal(t, []blockRange{{101, 110}}, h.takeFiltered())
	})
}

func TestController_RangeItFailsToLookAtIsAGapNeverLookedAtAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		covered := h.watch(t)
		h.readAt(100)
		h.start()
		defer h.controller.Stop()

		h.setFailing(true)
		h.readAt(150)
		h.scan()
		h.setFailing(false)
		h.readAt(160)
		h.scan()

		looked, gaps := covered.take()
		require.Equal(t, []blockRange{{151, 160}}, looked)
		require.Equal(t, []blockRange{{101, 150}}, gaps)
	})
}

func TestController_PausedBlocksAreLookedAtOnResume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		h.readAt(100)
		h.start()
		h.controller.Stop()

		time.Sleep(30 * time.Minute)
		h.readAt(400)
		h.start()
		defer h.controller.Stop()

		require.Equal(t, []blockRange{{101, 400}}, h.takeFiltered())
	})
}

// Every block after the first one balances were read at is either looked at
// once or reported as a gap, whatever fails and however detection pauses.
func TestController_EveryBlockIsLookedAtOnceOrIsAGap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newBlocksHarness(t)
		covered := h.watch(t)
		h.readAt(1000)
		h.start()

		block := uint64(1000)
		random := uint64(7)
		next := func(n uint64) uint64 {
			random = random*6364136223846793005 + 1442695040888963407
			return (random >> 33) % n
		}
		for range 200 {
			block += next(40)
			h.readAt(block)
			h.setFailing(next(4) == 0)
			if next(10) == 0 {
				h.controller.Stop()
				h.start()
				continue
			}
			h.scan()
		}
		h.setFailing(false)
		h.scan()
		h.controller.Stop()

		looked, gaps := covered.take()
		ranges := append(looked, gaps...)
		slices.SortFunc(ranges, func(a, b blockRange) int { return int(a.from) - int(b.from) })
		require.NotEmpty(t, gaps)
		next1001 := uint64(1001)
		for _, r := range ranges {
			require.Equal(t, next1001, r.from, "ranges must meet without holes or overlaps")
			require.LessOrEqual(t, r.from, r.to)
			next1001 = r.to + 1
		}
		require.Equal(t, block+1, next1001, "every block up to the last read one is covered")
	})
}
