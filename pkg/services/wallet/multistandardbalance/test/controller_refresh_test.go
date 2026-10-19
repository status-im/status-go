package multistandardbalance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"
	tokentypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
	"github.com/status-im/status-go/pkg/services/networks"
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
	dave  = types.Address{0xd0}

	usdc = common.Address{0x05}
	dai  = common.Address{0x06}
	weth = common.Address{0x07}
	// native stands for the native coin among the tokens a read covers.
	native = common.Address{}
)

// read is one FetchBalances call: when it was made, for which chain, and the
// tokens it read of each account, keyed by account address in hex.
type read struct {
	at       time.Duration
	chain    uint64
	balances map[string][]common.Address
}

type pair struct {
	account types.Address
	chain   uint64
}

type balanceOf struct {
	pair
	token common.Address
}

// fakeChangeSource records when it is scanned and tells what the test emits
// while it is watched.
type fakeChangeSource struct {
	start time.Time

	mu    sync.Mutex
	emit  func(refresh.Event)
	scans []time.Duration
}

func (s *fakeChangeSource) Watch(emit func(refresh.Event)) (stop func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emit = emit
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.emit = nil
	}
}

func (s *fakeChangeSource) Scan() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scans = append(s.scans, time.Since(s.start))
}

// tell emits ev as the source would, if it is watched.
func (s *fakeChangeSource) tell(ev refresh.Event) {
	s.mu.Lock()
	emit := s.emit
	s.mu.Unlock()
	if emit != nil {
		emit(ev)
	}
}

func (s *fakeChangeSource) takeScans() []time.Duration {
	synctest.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	scans := s.scans
	s.scans = nil
	return scans
}

// refreshHarness is a controller over a fake chain: its fetches answer with
// the balances the test sets, zero for the others, each at a new block. It
// records every fetch that reads anything.
type refreshHarness struct {
	controller *multistandardbalance.Controller
	walletFeed *event.Feed
	accounts   *pubsub.Publisher
	networks   *pubsub.Publisher
	source     *fakeChangeSource
	logs       *observer.ObservedLogs
	start      time.Time

	mu       sync.Mutex
	wallet   []types.Address
	chains   []uint64
	tokens   []common.Address
	held     map[pair][]common.Address
	balances map[balanceOf]*big.Int
	// failing are the chains whose fetches fail to start, erring those whose
	// results are errors.
	failing map[uint64]bool
	erring  map[uint64]bool
	block   int64
	reads   []read
}

func newRefreshHarness(t *testing.T) *refreshHarness {
	ctrl := gomock.NewController(t)
	fetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	heldTokensProvider := mock_multistandardbalance.NewMockHeldTokensProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)
	core, logs := observer.New(zapcore.WarnLevel)

	h := &refreshHarness{
		walletFeed: new(event.Feed),
		accounts:   pubsub.NewPublisher(),
		networks:   pubsub.NewPublisher(),
		logs:       logs,
		start:      time.Now(),
		source:     &fakeChangeSource{start: time.Now()},
		wallet:     []types.Address{alice, bob, carol},
		chains:     []uint64{1, 10, 56},
		tokens:     []common.Address{usdc, dai},
		held:       make(map[pair][]common.Address),
		balances:   make(map[balanceOf]*big.Int),
		failing:    make(map[uint64]bool),
		erring:     make(map[uint64]bool),
	}

	accountsProvider.EXPECT().GetWalletAddresses().DoAndReturn(func() ([]types.Address, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		return slices.Clone(h.wallet), nil
	}).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().DoAndReturn(func() ([]*params.Network, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		networks := make([]*params.Network, 0, len(h.chains))
		for _, chain := range h.chains {
			networks = append(networks, &params.Network{ChainID: chain})
		}
		return networks, nil
	}).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(h.networks).AnyTimes()
	tokenListProvider.EXPECT().GetTokenContractAddresses(gomock.Any()).DoAndReturn(func(uint64) ([]common.Address, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		return slices.Clone(h.tokens), nil
	}).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(gomock.Any(), gomock.Any()).
		Return([]multistandardbalance.CollectibleID{}, []multistandardbalance.CollectibleID{}, nil).AnyTimes()
	heldTokensProvider.EXPECT().GetHeldTokens(gomock.Any(), gomock.Any()).DoAndReturn(func(chain uint64, account common.Address) ([]common.Address, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		return slices.Clone(h.held[pair{types.Address(account), chain}]), nil
	}).AnyTimes()
	lastBlockManager.EXPECT().SetLatestBlockNumber(gomock.Any(), gomock.Any()).AnyTimes()
	fetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(h.fetch).AnyTimes()

	h.controller = multistandardbalance.NewController(
		multistandardbalance.DefaultControllerConfig(),
		multistandardbalance.NewStorageMemory(), fetcher, accountsProvider, h.accounts, networksProvider,
		tokenListProvider, collectibleListProvider, heldTokensProvider, h.source, lastBlockManager, h.walletFeed, zap.New(core))
	return h
}

func (h *refreshHarness) fetch(_ context.Context, chain uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failing[chain] {
		return nil, errors.New("fetch failed to start")
	}
	h.block++
	block := big.NewInt(h.block)
	var err error
	if h.erring[chain] {
		err = errors.New("fetch failed")
	}

	balances := make(map[string][]common.Address)
	results := make(chan multistandardfetcher.FetchResult, len(config.Native)+len(config.ERC20))
	for _, account := range config.Native {
		balances[account.Hex()] = append(balances[account.Hex()], native)
		results <- multistandardfetcher.FetchResult{ResultType: multistandardfetcher.ResultTypeNative, Result: multistandardfetcher.NativeResult{
			Account: account, Result: h.balance(account, chain, native), Err: err, AtBlockNumber: block,
		}}
	}
	for account, tokens := range config.ERC20 {
		answered := make(map[multistandardbalance.ContractAddress]*big.Int, len(tokens))
		for _, token := range tokens {
			balances[account.Hex()] = append(balances[account.Hex()], token)
			answered[token] = h.balance(account, chain, token)
		}
		results <- multistandardfetcher.FetchResult{ResultType: multistandardfetcher.ResultTypeERC20, Result: multistandardfetcher.ERC20Result{
			Account: account, Results: answered, Err: err, AtBlockNumber: block,
		}}
	}
	close(results)
	if len(balances) > 0 {
		h.reads = append(h.reads, read{at: time.Since(h.start), chain: chain, balances: balances})
	}
	return results, nil
}

func (h *refreshHarness) balance(account common.Address, chain uint64, token common.Address) *big.Int {
	if balance, ok := h.balances[balanceOf{pair{types.Address(account), chain}, token}]; ok {
		return new(big.Int).Set(balance)
	}
	return big.NewInt(0)
}

func (h *refreshHarness) setBalance(account types.Address, chain uint64, token common.Address, balance int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.balances[balanceOf{pair{account, chain}, token}] = big.NewInt(balance)
}

func (h *refreshHarness) setHeld(account types.Address, chain uint64, tokens ...common.Address) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.held[pair{account, chain}] = tokens
}

func (h *refreshHarness) setFailing(chain uint64, failing bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failing[chain] = failing
}

func (h *refreshHarness) setErring(chain uint64, erring bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.erring[chain] = erring
}

func (h *refreshHarness) addAccount(account types.Address) {
	h.mu.Lock()
	h.wallet = append(h.wallet, account)
	h.mu.Unlock()
	pubsub.Publish(h.accounts, accountsevent.AccountsAddedEvent{})
}

func (h *refreshHarness) addChain(chain uint64) {
	h.mu.Lock()
	h.chains = append(h.chains, chain)
	h.mu.Unlock()
	pubsub.Publish(h.networks, networks.EventActiveNetworksChanged{})
}

func (h *refreshHarness) addToken(token common.Address) {
	h.mu.Lock()
	h.tokens = append(h.tokens, token)
	h.mu.Unlock()
	h.walletFeed.Send(walletevent.Event{Type: walletevent.EventTokenListsUpdated})
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

// everything is a read of every token of every account on every chain at at.
func (h *refreshHarness) everything(at time.Duration) []read {
	h.mu.Lock()
	defer h.mu.Unlock()
	tokens := append([]common.Address{native}, h.tokens...)
	reads := make([]read, 0, len(h.chains))
	for _, chain := range h.chains {
		balances := make(map[string][]common.Address, len(h.wallet))
		for _, account := range h.wallet {
			balances[account.Hex()] = tokens
		}
		reads = append(reads, read{at: at, chain: chain, balances: balances})
	}
	return reads
}

// heldSet is a read of the native coin of every account on every chain at
// at, and of the tokens of tokens.
func (h *refreshHarness) heldSet(at time.Duration, tokens map[pair][]common.Address) []read {
	h.mu.Lock()
	defer h.mu.Unlock()
	reads := make([]read, 0, len(h.chains))
	for _, chain := range h.chains {
		balances := make(map[string][]common.Address, len(h.wallet))
		for _, account := range h.wallet {
			balances[account.Hex()] = append([]common.Address{native}, tokens[pair{account, chain}]...)
		}
		reads = append(reads, read{at: at, chain: chain, balances: balances})
	}
	return reads
}

// readsAt are the reads made at at.
func readsAt(reads []read, at time.Duration) []read {
	var selected []read
	for _, r := range reads {
		if r.at == at {
			selected = append(selected, r)
		}
	}
	return selected
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

func requireReads(t *testing.T, expected []read, actual []read) {
	t.Helper()
	normalize := func(reads []read) []read {
		reads = slices.Clone(reads)
		for _, r := range reads {
			for account, tokens := range r.balances {
				tokens = slices.Clone(tokens)
				slices.SortFunc(tokens, func(a, b common.Address) int { return bytes.Compare(a[:], b[:]) })
				r.balances[account] = tokens
			}
		}
		slices.SortFunc(reads, func(a, b read) int {
			if a.at != b.at {
				return int(a.at - b.at)
			}
			return int(a.chain) - int(b.chain)
		})
		return reads
	}
	require.Equal(t, normalize(expected), normalize(actual))
}

func (h *refreshHarness) sendWalletEvent(t *testing.T, eventType walletevent.EventType, payload any) {
	message, err := json.Marshal(payload)
	require.NoError(t, err)
	h.walletFeed.Send(walletevent.Event{Type: eventType, Message: string(message)})
}

func TestRefresh_ReadsEverythingOnStartAndTheHeldSetEveryPeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.setBalance(alice, 1, usdc, 5)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(4*time.Minute + time.Second)

		held := map[pair][]common.Address{{alice, 1}: {usdc}}
		requireReads(t, slices.Concat(h.everything(0), h.heldSet(2*time.Minute, held), h.heldSet(4*time.Minute, held)), h.takeReads())
	})
}

func TestRefresh_ReadsEverythingEveryHourOnScreen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(61 * time.Minute)

		reads := h.takeReads()
		require.Len(t, readTimes(reads), 31)
		requireReads(t, h.everything(time.Hour), readsAt(reads, time.Hour))
	})
}

func TestRefresh_HiddenBalancesReadTheHeldSetAndEverythingWhenBackOnScreenAfterAnHour(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.SetActive(false)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(61 * time.Minute)
		reads := h.takeReads()
		require.Equal(t, []time.Duration{0, 15 * time.Minute, 30 * time.Minute, 45 * time.Minute, time.Hour}, readTimes(reads))
		requireReads(t, h.heldSet(time.Hour, nil), readsAt(reads, time.Hour))

		h.controller.SetActive(true)

		requireReads(t, h.everything(61*time.Minute), h.takeReads())
	})
}

func TestRefresh_HeldSetHasTheTokensOfHeldTokensProvider(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.setHeld(bob, 10, dai)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(2*time.Minute + time.Second)

		requireReads(t, h.heldSet(2*time.Minute, map[pair][]common.Address{{bob, 10}: {dai}}), h.takeReads())
	})
}

func TestRefresh_TokenReadAsZeroLeavesTheHeldSet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.setBalance(alice, 1, usdc, 5)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(time.Minute)
		h.setBalance(alice, 1, usdc, 0)
		time.Sleep(3*time.Minute + time.Second)

		requireReads(t, slices.Concat(
			h.heldSet(2*time.Minute, map[pair][]common.Address{{alice, 1}: {usdc}}),
			h.heldSet(4*time.Minute, nil),
		), h.takeReads())
	})
}

func TestRefresh_AddedAccountIsReadInFullAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.addAccount(dave)
		time.Sleep(15 * time.Second)

		everything := []common.Address{native, usdc, dai}
		requireReads(t, []read{
			{at: 40 * time.Second, chain: 1, balances: map[string][]common.Address{dave.Hex(): everything}},
			{at: 40 * time.Second, chain: 10, balances: map[string][]common.Address{dave.Hex(): everything}},
			{at: 40 * time.Second, chain: 56, balances: map[string][]common.Address{dave.Hex(): everything}},
		}, h.takeReads())
	})
}

func TestRefresh_NewlyActiveChainIsReadInFullAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.addChain(8453)
		time.Sleep(15 * time.Second)

		everything := []common.Address{native, usdc, dai}
		requireReads(t, []read{{at: 40 * time.Second, chain: 8453, balances: map[string][]common.Address{
			alice.Hex(): everything, bob.Hex(): everything, carol.Hex(): everything,
		}}}, h.takeReads())
	})
}

func TestRefresh_TokensNewToTheListsAreReadAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.addToken(weth)
		time.Sleep(15 * time.Second)

		var reads []read
		for _, chain := range []uint64{1, 10, 56} {
			reads = append(reads, read{at: 40 * time.Second, chain: chain, balances: map[string][]common.Address{
				alice.Hex(): {weth}, bob.Hex(): {weth}, carol.Hex(): {weth},
			}})
		}
		requireReads(t, reads, h.takeReads())
	})
}

func TestRefresh_RequestedRefreshReadsTheHeldSetSoonAfterAFullRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.controller.TriggerFullFetch()
		time.Sleep(15 * time.Second)

		requireReads(t, h.heldSet(40*time.Second, nil), h.takeReads())
	})
}

func TestRefresh_RequestedRefreshReadsEverythingLongAfterAFullRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(11 * time.Minute)
		h.takeReads()
		h.controller.TriggerFullFetch()
		time.Sleep(15 * time.Second)

		requireReads(t, h.everything(11*time.Minute+10*time.Second), h.takeReads())
	})
}

func TestRefresh_EndedTransactionReadsItsTokensAndNativeCoinAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.sendWalletEvent(t, pendingtxtracker.EventPendingTransactionUpdate, pendingtxtracker.PendingTxUpdatePayload{
			Deleted: true,
			TxDetails: pendingtxtracker.TxDetails{SentTransactions: []*responses.RouterSentTransaction{{
				FromAddress: alice, ToAddress: bob, FromChain: 1, ToChain: 10,
				FromTokenKey: tokentypes.TokenKey(1, usdc), ToTokenKey: tokentypes.TokenKey(10, dai),
			}}},
		})
		time.Sleep(15 * time.Second)

		requireReads(t, []read{
			{at: 40 * time.Second, chain: 1, balances: map[string][]common.Address{alice.Hex(): {native, usdc}}},
			{at: 40 * time.Second, chain: 10, balances: map[string][]common.Address{bob.Hex(): {native, dai}}},
		}, h.takeReads())
	})
}

func TestRefresh_ConfirmedTransactionReadsItsTokensAtOnceInOneFetchPerChain(t *testing.T) {
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
				FromTokenKey: tokentypes.TokenKey(1, usdc), ToTokenKey: tokentypes.TokenKey(1, usdc),
			}},
		})

		requireReads(t, []read{{at: 30 * time.Second, chain: 1, balances: map[string][]common.Address{
			alice.Hex(): {native, usdc}, bob.Hex(): {native, usdc},
		}}}, h.takeReads())
	})
}

func TestRefresh_ConfirmedTransactionWithoutDetailsReadsEveryNativeCoinAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.sendWalletEvent(t, pendingtxtracker.EventPendingTransactionStatusChanged, pendingtxtracker.StatusChangedPayload{Status: ac.Success})

		requireReads(t, h.heldSet(30*time.Second, nil), h.takeReads())
	})
}

func TestRefresh_FullRefreshOfAChainThatFailedToStartIsRetriedAtTheNextPeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.setFailing(10, true)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		h.setFailing(10, false)
		time.Sleep(4*time.Minute + time.Second)

		reads := h.heldSet(2*time.Minute, nil)
		reads[1] = h.everything(2 * time.Minute)[1]
		requireReads(t, slices.Concat(reads, h.heldSet(4*time.Minute, nil)), h.takeReads())
	})
}

func TestRefresh_FullRefreshOfAChainWithFailedResultsIsRetriedAtTheNextPeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.setErring(56, true)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		h.setErring(56, false)
		time.Sleep(2*time.Minute + time.Second)

		reads := h.heldSet(2*time.Minute, nil)
		reads[2] = h.everything(2 * time.Minute)[2]
		requireReads(t, reads, h.takeReads())
	})
}

func TestRefresh_PartialReadsDoNotWarnAboutMissingBalances(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.setBalance(alice, 1, usdc, 5)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(2*time.Minute + time.Second)
		h.takeReads()

		require.Zero(t, h.logs.FilterMessageSnippet("missing").Len())
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

func TestRefresh_ResumeAfterAShortPauseReadsTheHeldSet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		time.Sleep(time.Minute)
		h.controller.Stop()
		h.takeReads()

		time.Sleep(10 * time.Minute)
		h.controller.Start()
		defer h.controller.Stop()

		requireReads(t, h.heldSet(11*time.Minute, nil), h.takeReads())
	})
}

func TestRefresh_ResumeAfterALongPauseReadsEverything(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		time.Sleep(time.Minute)
		h.controller.Stop()
		h.takeReads()

		time.Sleep(2 * time.Hour)
		h.controller.Start()
		defer h.controller.Stop()

		requireReads(t, h.everything(2*time.Hour+time.Minute), h.takeReads())
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

func TestRefresh_DetectedTransferReadsWhatMovedAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.source.tell(refresh.TransferDetected{Pair: refresh.Pair{Account: common.Address(alice), Chain: 1}, Contracts: []common.Address{dai}, Native: true})
		time.Sleep(15 * time.Second)

		requireReads(t, []read{{at: 40 * time.Second, chain: 1, balances: map[string][]common.Address{alice.Hex(): {native, dai}}}}, h.takeReads())
	})
}

func TestRefresh_DetectedTokenStaysHeldUntilReadAsZero(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		h.setErring(1, true)
		time.Sleep(30 * time.Second)
		h.source.tell(refresh.TransferDetected{Pair: refresh.Pair{Account: common.Address(alice), Chain: 1}, Contracts: []common.Address{weth}})
		time.Sleep(15 * time.Second)
		h.setErring(1, false)
		time.Sleep(3*time.Minute + 30*time.Second)

		requireReads(t, slices.Concat(
			[]read{{at: 40 * time.Second, chain: 1, balances: map[string][]common.Address{alice.Hex(): {weth}}}},
			h.heldSet(2*time.Minute, map[pair][]common.Address{{alice, 1}: {weth}}),
			h.heldSet(4*time.Minute, nil),
		), h.takeReads())
	})
}

func TestRefresh_DetectionGapReadsItsChainInFullAfterDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.takeReads()

		time.Sleep(30 * time.Second)
		h.source.tell(refresh.DetectionGap{Chain: 10})
		time.Sleep(15 * time.Second)

		requireReads(t, []read{h.everything(40 * time.Second)[1]}, h.takeReads())
	})
}

func TestRefresh_ScansForChangesWithEveryPeriodicRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()

		time.Sleep(30 * time.Second)
		h.source.tell(refresh.TransferDetected{Pair: refresh.Pair{Account: common.Address(alice), Chain: 1}, Native: true})
		time.Sleep(4*time.Minute - 30*time.Second + time.Second)

		require.Equal(t, []time.Duration{0, 2 * time.Minute, 4 * time.Minute}, h.source.takeScans())
	})
}

func TestRefresh_RequestedRefreshScansForChangesAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		defer h.controller.Stop()
		h.source.takeScans()

		time.Sleep(30 * time.Second)
		h.controller.TriggerFullFetch()

		require.Equal(t, []time.Duration{30 * time.Second}, h.source.takeScans())
	})
}

func TestRefresh_StoppedControllerHearsNoChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newRefreshHarness(t)
		h.controller.Start()
		h.controller.Stop()
		h.takeReads()

		h.source.tell(refresh.DetectionGap{Chain: 10})
		time.Sleep(time.Minute)

		require.Empty(t, h.takeReads())
	})
}
