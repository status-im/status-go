package multistandardbalance_test

import (
	"context"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	mock_multistandardbalance "github.com/status-im/status-go/pkg/services/wallet/multistandardbalance/mock"
)

const (
	pollDebounce = 10 * time.Millisecond
	pollActive   = 100 * time.Millisecond
	pollIdle     = time.Hour
)

// newPollingController is a controller over one account on one chain whose
// balances were fetched minutes ago, counting the fetches that ask for any.
func newPollingController(t *testing.T) (*multistandardbalance.Controller, *atomic.Int32) {
	ctrl := gomock.NewController(t)
	storage := mock_multistandardbalance.NewMockStorage(ctrl)
	fetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)

	address := types.Address{0x11}
	key := multistandardbalance.BalancesKey{Account: common.BytesToAddress(address.Bytes()), ChainID: 1}
	stale := multistandardbalance.State{AtBlockNumber: big.NewInt(100), FetchedAt: time.Now().Unix() - 600}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{address}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{{ChainID: 1}}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	storage.EXPECT().GetNativeBalance(gomock.Any(), key).Return(big.NewInt(1), stale, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key).Return(map[multistandardbalance.ContractAddress]*big.Int{}, stale, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key).Return(map[multistandardbalance.ContractAddress]*big.Int{}, stale, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, stale, nil).AnyTimes()
	storage.EXPECT().ClearMissingAccounts(gomock.Any(), gomock.Any()).AnyTimes()
	storage.EXPECT().ClearMissingChains(gomock.Any(), gomock.Any()).AnyTimes()
	tokenListProvider.EXPECT().GetTokenContractAddresses(uint64(1)).Return([]common.Address{}, nil).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(gomock.Any(), gomock.Any()).
		Return([]multistandardbalance.CollectibleID{}, []multistandardbalance.CollectibleID{}, nil).AnyTimes()

	var fetches atomic.Int32
	fetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint64, config interface{}) (<-chan interface{}, error) {
			if c := config.(multistandardfetcher.FetchConfig); len(c.Native)+len(c.ERC20)+len(c.ERC721)+len(c.ERC1155) > 0 {
				fetches.Add(1)
			}
			ch := make(chan interface{})
			close(ch)
			return ch, nil
		}).AnyTimes()

	controller := multistandardbalance.NewController(
		multistandardbalance.ControllerConfig{FetchDebounceTime: pollDebounce, FetchPeriod: pollActive, IdleFetchPeriod: pollIdle},
		storage, fetcher, accountsProvider, pubsub.NewPublisher(), networksProvider,
		tokenListProvider, collectibleListProvider, lastBlockManager, nil, zap.NewNop())
	return controller, &fetches
}

func TestController_PollsAtTheActivePeriodByDefault(t *testing.T) {
	controller, fetches := newPollingController(t)
	controller.Start()
	defer controller.Stop()

	time.Sleep(5 * pollActive)

	require.GreaterOrEqual(t, fetches.Load(), int32(3))
}

func TestController_PollsAtTheIdlePeriodWhileInactive(t *testing.T) {
	controller, fetches := newPollingController(t)
	controller.SetActive(false)
	controller.Start()
	defer controller.Stop()
	require.Eventually(t, func() bool { return fetches.Load() == 1 }, 10*pollActive, pollDebounce, "the start fetch")

	time.Sleep(5 * pollActive)

	require.Equal(t, int32(1), fetches.Load(), "no periodic fetch within the idle period")
}

func TestController_FetchesAtOnceWhenActiveAfterTheActivePeriod(t *testing.T) {
	controller, fetches := newPollingController(t)
	controller.SetActive(false)
	controller.Start()
	defer controller.Stop()
	require.Eventually(t, func() bool { return fetches.Load() == 1 }, 10*pollActive, pollDebounce, "the start fetch")
	time.Sleep(2 * pollActive)

	controller.SetActive(true)

	require.Eventually(t, func() bool { return fetches.Load() == 2 }, pollActive/2, pollDebounce/2,
		"balances on screen again are refreshed without waiting for the next period")
	time.Sleep(4 * pollActive)
	require.GreaterOrEqual(t, fetches.Load(), int32(4), "and polled at the active period from then on")
}

func TestController_KeepsBalancesWithinTheIdlePeriodOnOtherTriggers(t *testing.T) {
	controller, fetches := newPollingController(t)
	controller.SetActive(false)
	controller.Start()
	defer controller.Stop()
	require.Eventually(t, func() bool { return fetches.Load() == 1 }, 10*pollActive, pollDebounce, "the start fetch")

	controller.TriggerFetchWithConfig(multistandardbalance.FetchConfig{})
	time.Sleep(5 * pollDebounce)

	require.Equal(t, int32(1), fetches.Load(), "balances fetched minutes ago are fresh while inactive")
}

func TestController_DoesNotFetchWhenActiveAgainSoonAfterAFetch(t *testing.T) {
	controller, fetches := newPollingController(t)
	controller.Start()
	defer controller.Stop()
	require.Eventually(t, func() bool { return fetches.Load() == 1 }, pollActive/2, pollDebounce/2, "the start fetch")

	controller.SetActive(false)
	controller.SetActive(true)
	time.Sleep(pollActive / 2)

	require.Equal(t, int32(1), fetches.Load(), "the last fetch is still fresh")
}
