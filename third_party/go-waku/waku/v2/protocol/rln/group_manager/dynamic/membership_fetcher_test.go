package dynamic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
)

func TestFetchingLogic(t *testing.T) {
	client := NewMockClient(t, "membership_fetcher.json")

	rlnContract, err := contracts.NewRLN(common.Address{}, client)
	require.NoError(t, err)
	rlnInstance, err := rln.NewRLN()
	require.NoError(t, err)

	rootTracker := group_manager.NewMerkleRootTracker(1, rlnInstance)

	mf := MembershipFetcher{
		web3Config: &web3.Config{
			RLNContract: web3.RLNContract{
				RLN: rlnContract,
			},
			ETHClient: client,
		},
		rln:         rlnInstance,
		log:         utils.Logger(),
		rootTracker: rootTracker,
	}

	counts := []int{}
	mockFn := func(events []*contracts.RLNMemberRegistered, latestProcessedBlock uint64) error {
		counts = append(counts, len(events))
		return nil
	}
	// check if more than 10k error is handled or not.
	client.SetErrorOnBlock(5007, fmt.Errorf("query returned more than 10000 results"), 2)
	// loadOldEvents will check till 10010
	client.SetLatestBlockNumber(10010)
	// watchNewEvents will check till 10012
	ctx, cancel := context.WithCancel(context.Background())
	if err := mf.HandleGroupUpdates(ctx, mockFn); err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			if client.latestBlockNum.Load() == 10012 {
				cancel()
				return
			}
			time.Sleep(time.Second)
		}
	}()
	mf.Stop()
	// sleep so that watchNewEvents can finish
	// check whether all the events are fetched or not.
	require.Equal(t, counts, []int{1, 3, 2, 1, 1})
}
