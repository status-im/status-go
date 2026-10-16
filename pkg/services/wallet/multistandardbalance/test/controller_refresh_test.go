package multistandardbalance_test

import (
	"context"
	"encoding/json"
	"math/big"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	mock_multistandardbalance "github.com/status-im/status-go/pkg/services/wallet/multistandardbalance/mock"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/pkg/services/wallet/responses"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

var (
	alice = types.Address{0xa1}
	bob   = types.Address{0xb0}
	carol = types.Address{0xc0}
	usdc  = common.Address{0x05}
)

var chains = []uint64{1, 10, 56}

var nativeAndERC20 = []multistandardfetcher.ResultType{multistandardfetcher.ResultTypeNative, multistandardfetcher.ResultTypeERC20}

// read is one FetchBalances call: when it was made, for which chain, and the
// kinds each account was read for.
type read struct {
	at    time.Duration
	chain uint64
	// kinds are keyed by account address in hex.
	kinds map[string][]multistandardfetcher.ResultType
}

// refreshHarness is a controller over three accounts on three chains whose
// balance fetches return nothing, recording every fetch that reads anything.
type refreshHarness struct {
	controller *multistandardbalance.Controller
	storage    *multistandardbalance.StorageMemory
	walletFeed *event.Feed
	accounts   *pubsub.Publisher
	start      time.Time

	mu    sync.Mutex
	reads []read
}

func newRefreshHarness(t *testing.T) *refreshHarness {
	ctrl := gomock.NewController(t)
	fetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)

	h := &refreshHarness{
		storage:    multistandardbalance.NewStorageMemory(),
		walletFeed: new(event.Feed),
		accounts:   pubsub.NewPublisher(),
		start:      time.Now(),
	}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{alice, bob, carol}, nil).AnyTimes()
	networks := make([]*params.Network, 0, len(chains))
	for _, chain := range chains {
		networks = append(networks, &params.Network{ChainID: chain})
	}
	networksProvider.EXPECT().GetActiveNetworks().Return(networks, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	tokenListProvider.EXPECT().GetTokenContractAddresses(gomock.Any()).Return([]common.Address{usdc}, nil).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(gomock.Any(), gomock.Any()).
		Return([]multistandardbalance.CollectibleID{}, []multistandardbalance.CollectibleID{}, nil).AnyTimes()
	fetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, chain uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
			h.record(chain, config)
			ch := make(chan multistandardfetcher.FetchResult)
			close(ch)
			return ch, nil
		}).AnyTimes()

	h.controller = multistandardbalance.NewController(
		multistandardbalance.DefaultControllerConfig(),
		h.storage, fetcher, accountsProvider, h.accounts, networksProvider,
		tokenListProvider, collectibleListProvider, lastBlockManager, h.walletFeed, zap.NewNop())
	return h
}

func (h *refreshHarness) record(chain uint64, config multistandardfetcher.FetchConfig) {
	kinds := make(map[string][]multistandardfetcher.ResultType)
	for _, account := range config.Native {
		kinds[account.Hex()] = append(kinds[account.Hex()], multistandardfetcher.ResultTypeNative)
	}
	for account := range config.ERC20 {
		kinds[account.Hex()] = append(kinds[account.Hex()], multistandardfetcher.ResultTypeERC20)
	}
	if len(kinds) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reads = append(h.reads, read{at: time.Since(h.start), chain: chain, kinds: kinds})
}

// takeReads returns the reads made so far and forgets them.
func (h *refreshHarness) takeReads() []read {
	synctest.Wait()
	h.mu.Lock()
	defer h.mu.Unlock()
	reads := h.reads
	h.reads = nil
	return reads
}

// readTimes are the times reads were made at, once each whatever the chain.
func readTimes(reads []read) []time.Duration {
	var times []time.Duration
	for _, r := range reads {
		if !slices.Contains(times, r.at) {
			times = append(times, r.at)
		}
	}
	return times
}

func everything(at time.Duration) []read {
	reads := make([]read, 0, len(chains))
	for _, chain := range chains {
		reads = append(reads, read{at: at, chain: chain, kinds: map[string][]multistandardfetcher.ResultType{
			alice.Hex(): nativeAndERC20, bob.Hex(): nativeAndERC20, carol.Hex(): nativeAndERC20,
		}})
	}
	return reads
}

func requireReads(t *testing.T, expected []read, actual []read) {
	t.Helper()
	byTimeAndChain := func(a, b read) int {
		if a.at != b.at {
			return int(a.at - b.at)
		}
		return int(a.chain) - int(b.chain)
	}
	slices.SortFunc(expected, byTimeAndChain)
	slices.SortFunc(actual, byTimeAndChain)
	require.Equal(t, expected, actual)
}

// storeAt stores balances of every kind of account on chain as read at fetchedAt.
func (h *refreshHarness) storeAt(t *testing.T, account types.Address, chain uint64, fetchedAt time.Time) {
	ctx := context.Background()
	key := multistandardbalance.BalancesKey{Account: common.Address(account), ChainID: chain}
	state := multistandardbalance.State{AtBlockNumber: big.NewInt(fetchedAt.UnixNano()), FetchedAt: fetchedAt.Unix()}
	_, _, err := h.storage.UpdateNativeBalance(ctx, key, big.NewInt(1), state)
	require.NoError(t, err)
	_, _, err = h.storage.UpdateERC20Balances(ctx, key, map[multistandardbalance.ContractAddress]*big.Int{usdc: big.NewInt(1)}, state)
	require.NoError(t, err)
	_, _, err = h.storage.UpdateERC721Balances(ctx, key, map[multistandardbalance.ContractAddress]*big.Int{}, state)
	require.NoError(t, err)
	_, _, err = h.storage.UpdateERC1155Balances(ctx, key, map[multistandardbalance.HashableCollectibleID]*big.Int{}, state)
	require.NoError(t, err)
}

func (h *refreshHarness) sendWalletEvent(t *testing.T, eventType walletevent.EventType, payload any) {
	message, err := json.Marshal(payload)
	require.NoError(t, err)
	h.walletFeed.Send(walletevent.Event{Type: eventType, Message: string(message)})
}

func TestRefresh_ReadsEverythingOnStartAndEveryPeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(4*time.Minute + time.Second)

		requireReads(t, slices.Concat(everything(0), everything(2*time.Minute), everything(4*time.Minute)), h.takeReads())
	})
}

func TestRefresh_ReadsHiddenBalancesEveryHiddenPeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.SetActive(false)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(31 * time.Minute)

		require.Equal(t, []time.Duration{0, 15 * time.Minute, 30 * time.Minute}, readTimes(h.takeReads()))
	})
}

func TestRefresh_BalancesBackOnScreenAreReadAtOnceWhenStale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.SetActive(false)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(5 * time.Minute)
		h.controller.SetActive(true)
		time.Sleep(2*time.Minute + time.Second)

		require.Equal(t, []time.Duration{0, 5 * time.Minute, 7 * time.Minute}, readTimes(h.takeReads()))
	})
}

func TestRefresh_RequestedRefreshReadsEverythingAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.controller.TriggerFullFetch()
		time.Sleep(15 * time.Second)

		requireReads(t, everything(40*time.Second), h.takeReads())
	})
}

func TestRefresh_TokenListsUpdateReadsEverythingAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.walletFeed.Send(walletevent.Event{Type: walletevent.EventTokenListsUpdated})
		time.Sleep(15 * time.Second)

		requireReads(t, everything(40*time.Second), h.takeReads())
	})
}

func TestRefresh_AddedAccountsReadBalancesOlderThanThePeriodAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		for _, account := range []types.Address{alice, bob, carol} {
			for _, chain := range chains {
				fetchedAt := time.Now()
				if account == carol && chain == 10 {
					fetchedAt = fetchedAt.Add(-10 * time.Minute)
				}
				h.storeAt(t, account, chain, fetchedAt)
			}
		}
		pubsub.Publish(h.accounts, accountsevent.AccountsAddedEvent{})
		time.Sleep(15 * time.Second)

		requireReads(t, []read{{at: 40 * time.Second, chain: 10, kinds: map[string][]multistandardfetcher.ResultType{
			carol.Hex(): nativeAndERC20,
		}}}, h.takeReads())
	})
}

func TestRefresh_EndedTransactionReadsItsAccountsOnItsChainsAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.sendWalletEvent(t, pendingtxtracker.EventPendingTransactionUpdate, pendingtxtracker.PendingTxUpdatePayload{
			Deleted: true,
			TxDetails: pendingtxtracker.TxDetails{SentTransactions: []*responses.RouterSentTransaction{
				{FromAddress: alice, ToAddress: bob, FromChain: 1, ToChain: 10},
			}},
		})
		time.Sleep(15 * time.Second)

		aliceAndBob := map[string][]multistandardfetcher.ResultType{alice.Hex(): nativeAndERC20, bob.Hex(): nativeAndERC20}
		requireReads(t, []read{
			{at: 40 * time.Second, chain: 1, kinds: aliceAndBob},
			{at: 40 * time.Second, chain: 10, kinds: aliceAndBob},
		}, h.takeReads())
	})
}

func TestRefresh_ConfirmedTransactionReadsItsBalancesAtOnceInOneFetchPerChain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.sendWalletEvent(t, pendingtxtracker.EventPendingTransactionStatusChanged, pendingtxtracker.StatusChangedPayload{
			Status: ac.Success,
			TxDetails: pendingtxtracker.TxDetails{SendDetails: &responses.SendDetails{
				FromAddress: alice, FromChain: 1, ToAddress: bob, ToChain: 1, SendType: int(sendtype.Transfer),
			}},
		})

		requireReads(t, []read{{at: 30 * time.Second, chain: 1, kinds: map[string][]multistandardfetcher.ResultType{
			alice.Hex(): nativeAndERC20, bob.Hex(): nativeAndERC20,
		}}}, h.takeReads())
	})
}

func TestRefresh_ConfirmedTransactionWithoutDetailsReadsEverythingAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.sendWalletEvent(t, pendingtxtracker.EventPendingTransactionStatusChanged, pendingtxtracker.StatusChangedPayload{Status: ac.Success})

		requireReads(t, everything(30*time.Second), h.takeReads())
	})
}

func TestRefresh_StopDropsPendingReads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.controller.TriggerFullFetch()
		time.Sleep(5 * time.Second)
		h.controller.Stop()
		time.Sleep(5 * time.Minute)

		require.Empty(t, h.takeReads())
	})
}

func TestRefresh_RestartReadsEverythingAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		time.Sleep(time.Minute)
		h.controller.Stop()
		h.takeReads()

		time.Sleep(time.Minute)
		h.controller.Start()
		defer h.controller.Stop()

		requireReads(t, everything(2*time.Minute), h.takeReads())
	})
}

func TestRefresh_BalancesBackOnScreenSoonAfterAReadAreNotReadAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(30 * time.Second)
		h.controller.SetActive(false)
		h.controller.SetActive(true)
		time.Sleep(time.Minute)

		require.Equal(t, []time.Duration{0}, readTimes(h.takeReads()))
	})
}

func TestRefresh_HiddenBalancesReadMinutesAgoAreFreshWhenAccountsChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.SetActive(false)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		for _, account := range []types.Address{alice, bob, carol} {
			for _, chain := range chains {
				h.storeAt(t, account, chain, time.Now().Add(-10*time.Minute))
			}
		}
		pubsub.Publish(h.accounts, accountsevent.AccountsAddedEvent{})
		time.Sleep(time.Minute)

		require.Empty(t, h.takeReads())
	})
}
