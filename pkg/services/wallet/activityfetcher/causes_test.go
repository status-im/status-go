package activityfetcher

import (
	"math/big"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc1155"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc20"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"
)

var (
	alice     = gethcommon.HexToAddress("0xa1")
	bob       = gethcommon.HexToAddress("0xb0")
	stranger  = gethcommon.HexToAddress("0x5e")
	usdc      = gethcommon.HexToAddress("0xc0")
	spamToken = gethcommon.HexToAddress("0x5a")
	nft       = gethcommon.HexToAddress("0x7f")
)

const mainnet = uint64(1)

type fetchLog struct {
	mu      sync.Mutex
	fetches [][]fetcherID
}

func (l *fetchLog) record(ids []fetcherID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fetches = append(l.fetches, ids)
}

func (l *fetchLog) get() [][]fetcherID {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fetches
}

func TestCauses_GatheredIntoOneFetchAfterTheWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var log fetchLog
		c := newCauses(log.record)
		defer c.stop()

		c.add(fetcherID{alice, mainnet})
		time.Sleep(causeWindow / 2)
		c.add(fetcherID{alice, mainnet}, fetcherID{bob, mainnet})
		synctest.Wait()
		require.Empty(t, log.get())

		time.Sleep(causeWindow / 2)
		synctest.Wait()
		require.Equal(t, [][]fetcherID{{{alice, mainnet}, {bob, mainnet}}}, log.get())

		c.add(fetcherID{alice, mainnet})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Len(t, log.get(), 2)
	})
}

func TestCauses_FailedFetchAskedAgainAtTheNextBalanceRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var log fetchLog
		c := newCauses(log.record)
		defer c.stop()

		c.balanceRead(fetcherID{alice, mainnet})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Empty(t, log.get(), "a balance read alone is no cause")

		c.fetchFailed(fetcherID{alice, mainnet})
		c.balanceRead(fetcherID{bob, mainnet})
		c.balanceRead(fetcherID{alice, mainnet})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Equal(t, [][]fetcherID{{{alice, mainnet}}}, log.get())

		c.balanceRead(fetcherID{alice, mainnet})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Len(t, log.get(), 1, "asked again once per failure")

		c.fetchFailed(fetcherID{alice, mainnet})
		c.fetchSucceeded(fetcherID{alice, mainnet})
		c.balanceRead(fetcherID{alice, mainnet})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Len(t, log.get(), 1, "a later success clears the failure")
	})
}

func TestCauses_StopDropsWhatWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var log fetchLog
		c := newCauses(log.record)
		c.add(fetcherID{alice, mainnet})
		c.stop()
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Empty(t, log.get())
	})
}

func erc20Log(token, from, to gethcommon.Address) eventlog.Event {
	return eventlog.Event{
		EventKey: eventlog.ERC20Transfer,
		Unpacked: erc20.Erc20Transfer{From: from, To: to, Value: big.NewInt(1), Raw: types.Log{Address: token}},
	}
}

func knownOnly(tokens ...gethcommon.Address) func(gethcommon.Address) bool {
	return func(token gethcommon.Address) bool {
		for _, known := range tokens {
			if known == token {
				return true
			}
		}
		return false
	}
}

func TestTransferCauses(t *testing.T) {
	accounts := []gethcommon.Address{alice, bob}
	tests := []struct {
		name string
		logs []eventlog.Event
		want []fetcherID
	}{
		{
			name: "known token, both sides are accounts",
			logs: []eventlog.Event{erc20Log(usdc, alice, bob)},
			want: []fetcherID{{alice, mainnet}, {bob, mainnet}},
		},
		{
			name: "token not on the lists is spam",
			logs: []eventlog.Event{erc20Log(spamToken, stranger, alice)},
		},
		{
			name: "native transfer log",
			logs: []eventlog.Event{erc20Log(refresh.NativeTransferLogAddress, stranger, alice)},
			want: []fetcherID{{alice, mainnet}},
		},
		{
			name: "every collectible",
			logs: []eventlog.Event{
				{EventKey: eventlog.ERC721Transfer, Unpacked: erc721.Erc721Transfer{From: stranger, To: alice, TokenId: big.NewInt(1), Raw: types.Log{Address: nft}}},
				{EventKey: eventlog.ERC1155TransferSingle, Unpacked: erc1155.Erc1155TransferSingle{From: bob, To: stranger, Raw: types.Log{Address: nft}}},
			},
			want: []fetcherID{{alice, mainnet}, {bob, mainnet}},
		},
		{
			name: "collectible batch",
			logs: []eventlog.Event{
				{EventKey: eventlog.ERC1155TransferBatch, Unpacked: erc1155.Erc1155TransferBatch{From: stranger, To: bob, Raw: types.Log{Address: nft}}},
			},
			want: []fetcherID{{bob, mainnet}},
		},
		{
			name: "one fetch per account",
			logs: []eventlog.Event{erc20Log(usdc, stranger, alice), erc20Log(refresh.NativeTransferLogAddress, stranger, alice)},
			want: []fetcherID{{alice, mainnet}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := transferdetector.EventTransferDetectionFinished{ChainID: mainnet, Accounts: accounts, Events: tt.logs}
			require.ElementsMatch(t, tt.want, transferCauses(ev, knownOnly(usdc)))
		})
	}
}

type fakeKnownTokens []gethcommon.Address

func (f fakeKnownTokens) GetTokenContractAddresses(uint64) ([]gethcommon.Address, error) {
	return f, nil
}

// watching runs the cause watchers of a service on the given publishers and
// records the fetches they ask for.
func watching(t *testing.T, test func(s *Service, balances, detection *pubsub.Publisher, log *fetchLog)) {
	synctest.Test(t, func(t *testing.T) {
		var log fetchLog
		balances, detection := pubsub.NewPublisher(), pubsub.NewPublisher()
		s := &Service{
			logger:                        zap.NewNop(),
			multistandardBalancePublisher: balances,
			transferDetectorPublisher:     detection,
			knownTokens:                   fakeKnownTokens{usdc},
			stopCh:                        make(chan struct{}),
		}
		s.causes = newCauses(log.record)
		s.startBalanceChangeWatcher()
		s.startTransferDetectionWatcher()

		test(s, balances, detection, &log)

		close(s.stopCh)
		s.causes.stop()
		synctest.Wait()
	})
}

func balanceRead(account gethcommon.Address, resultType multistandardfetcher.ResultType, changed bool) multistandardbalance.EventBalanceFetchFinished {
	return multistandardbalance.EventBalanceFetchFinished{
		Key:            multistandardbalance.BalancesKey{Account: account, ChainID: mainnet},
		ResultType:     resultType,
		BalanceChanged: changed,
		OldState:       multistandardbalance.State{AtBlockNumber: big.NewInt(100)},
		NewState:       multistandardbalance.State{AtBlockNumber: big.NewInt(101)},
	}
}

func TestService_TokenBalanceChangeIsNoCause(t *testing.T) {
	watching(t, func(_ *Service, balances, _ *pubsub.Publisher, log *fetchLog) {
		// A rebasing token (aToken) changes at every held refresh without a transfer.
		for range 5 {
			pubsub.Publish(balances, balanceRead(alice, multistandardfetcher.ResultTypeERC20, true))
			pubsub.Publish(balances, balanceRead(alice, multistandardfetcher.ResultTypeNative, false))
			time.Sleep(2 * time.Minute)
			synctest.Wait()
		}
		require.Empty(t, log.get())
	})
}

func TestService_NativeBalanceChangeIsACause(t *testing.T) {
	watching(t, func(_ *Service, balances, _ *pubsub.Publisher, log *fetchLog) {
		pubsub.Publish(balances, balanceRead(alice, multistandardfetcher.ResultTypeNative, true))
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Equal(t, [][]fetcherID{{{alice, mainnet}}}, log.get())
	})
}

func TestService_FirstNativeReadIsNoCause(t *testing.T) {
	watching(t, func(_ *Service, balances, _ *pubsub.Publisher, log *fetchLog) {
		first := balanceRead(alice, multistandardfetcher.ResultTypeNative, true)
		first.OldState = multistandardbalance.State{}
		pubsub.Publish(balances, first)
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Empty(t, log.get())
	})
}

func TestService_TransferAndNativeChangeOfOneAccountFetchOnce(t *testing.T) {
	watching(t, func(_ *Service, balances, detection *pubsub.Publisher, log *fetchLog) {
		pubsub.Publish(detection, transferdetector.EventTransferDetectionFinished{
			ChainID:  mainnet,
			Accounts: []gethcommon.Address{alice, bob},
			Events:   []eventlog.Event{erc20Log(refresh.NativeTransferLogAddress, stranger, alice)},
		})
		synctest.Wait()
		pubsub.Publish(balances, balanceRead(alice, multistandardfetcher.ResultTypeNative, true))
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Equal(t, [][]fetcherID{{{alice, mainnet}}}, log.get())
	})
}

func TestService_SpamTransferIsNoCause(t *testing.T) {
	watching(t, func(_ *Service, _, detection *pubsub.Publisher, log *fetchLog) {
		pubsub.Publish(detection, transferdetector.EventTransferDetectionFinished{
			ChainID:  mainnet,
			Accounts: []gethcommon.Address{alice},
			Events:   []eventlog.Event{erc20Log(spamToken, stranger, alice)},
		})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Empty(t, log.get())
	})
}

func TestService_DetectionGapFetchesEveryAccountOfTheChain(t *testing.T) {
	watching(t, func(_ *Service, _, detection *pubsub.Publisher, log *fetchLog) {
		pubsub.Publish(detection, transferdetector.EventTransferDetectionError{
			ChainID:  mainnet,
			Accounts: []gethcommon.Address{alice, bob},
		})
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Equal(t, [][]fetcherID{{{alice, mainnet}, {bob, mainnet}}}, log.get())
	})
}

func TestService_FailedFetchAskedAgainAtTheNextNativeRead(t *testing.T) {
	watching(t, func(s *Service, balances, _ *pubsub.Publisher, log *fetchLog) {
		s.causes.fetchFailed(fetcherID{alice, mainnet})
		pubsub.Publish(balances, balanceRead(alice, multistandardfetcher.ResultTypeERC20, false))
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Empty(t, log.get(), "only the native read comes with every balance refresh")

		pubsub.Publish(balances, balanceRead(alice, multistandardfetcher.ResultTypeNative, false))
		time.Sleep(causeWindow)
		synctest.Wait()
		require.Equal(t, [][]fetcherID{{{alice, mainnet}}}, log.get())
	})
}
