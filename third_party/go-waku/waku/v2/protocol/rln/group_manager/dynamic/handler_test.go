package dynamic

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/contracts"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/group_manager"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln/web3"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/go-zerokit-rln/rln"
)

func eventBuilder(blockNumber uint64, removed bool, idCommitment int64, index int64) *contracts.RLNMemberRegistered {
	return &contracts.RLNMemberRegistered{
		Raw: types.Log{
			BlockNumber: blockNumber,
			Removed:     removed,
		},
		Index:        big.NewInt(index),
		IdCommitment: big.NewInt(idCommitment),
	}
}

func TestHandler(t *testing.T) {
	// Create a RLN instance
	rlnInstance, err := rln.NewRLN()
	require.NoError(t, err)

	rootTracker := group_manager.NewMerkleRootTracker(5, rlnInstance)

	_, cancel := context.WithCancel(context.TODO())
	defer cancel()

	gm := &DynamicGroupManager{
		MembershipFetcher: NewMembershipFetcher(
			&web3.Config{
				ChainID: big.NewInt(1),
			},
			rlnInstance,
			rootTracker,
			utils.Logger(),
		),
		cancel: cancel,

		metrics: newMetrics(prometheus.DefaultRegisterer),
	}

	root0 := [32]byte{62, 31, 25, 34, 223, 182, 113, 211, 249, 18, 247, 234, 70, 30, 10, 136, 238, 132, 143, 221, 225, 43, 108, 24, 171, 26, 210, 197, 106, 231, 52, 33}
	roots := gm.rootTracker.Roots()
	require.Len(t, roots, 1)
	require.Equal(t, roots[0], root0)

	events := []*contracts.RLNMemberRegistered{eventBuilder(1, false, 0xaaaa, 1)}

	err = gm.handler(events, 1)
	require.NoError(t, err)

	roots = gm.rootTracker.Roots()

	require.Len(t, roots, 2)
	require.Equal(t, root0, roots[0])
	require.Equal(t, [32]byte{0x1c, 0xe4, 0x6b, 0x9a, 0xb2, 0x54, 0xa1, 0xb0, 0x2, 0x77, 0xbf, 0xc0, 0xf6, 0x27, 0x38, 0x38, 0x8f, 0xc8, 0x6a, 0x3d, 0x18, 0xe3, 0x1a, 0xd, 0xdd, 0xb8, 0xe5, 0x38, 0xf5, 0x9e, 0xc3, 0x16}, roots[1])

	events = []*contracts.RLNMemberRegistered{
		eventBuilder(1, false, 0xbbbb, 2),
		eventBuilder(2, false, 0xcccc, 3),
		eventBuilder(3, false, 0xdddd, 4),
		eventBuilder(4, false, 0xeeee, 5),
	}

	err = gm.handler(events, 4)
	require.NoError(t, err)

	// Root[1] should become [0]
	roots = gm.rootTracker.Roots()
	require.Len(t, roots, 5)
	require.Equal(t, [32]byte{0x1c, 0xe4, 0x6b, 0x9a, 0xb2, 0x54, 0xa1, 0xb0, 0x2, 0x77, 0xbf, 0xc0, 0xf6, 0x27, 0x38, 0x38, 0x8f, 0xc8, 0x6a, 0x3d, 0x18, 0xe3, 0x1a, 0xd, 0xdd, 0xb8, 0xe5, 0x38, 0xf5, 0x9e, 0xc3, 0x16}, roots[0])
	require.Len(t, rootTracker.Buffer(), 1)
	require.Equal(t, root0, rootTracker.Buffer()[0])

	// We detect a fork
	//
	// [0] -> [1] -> [2] -> [3] -> [4]   Our chain
	//                \
	//                 \-->              Real chain
	// We should restore the valid roots from the buffer at the state the moment the chain forked
	// In this case, just adding the original merkle root from empty tree
	validRootsBeforeFork := roots[0:3]
	events = []*contracts.RLNMemberRegistered{
		eventBuilder(3, true, 0xdddd, 4),
		eventBuilder(3, false, 0xdddd, 4),
		eventBuilder(3, false, 0xeeee, 5),
	}

	err = gm.handler(events, 3)
	require.NoError(t, err)

	roots = gm.rootTracker.Roots()
	require.Len(t, roots, 5)
	require.Equal(t, roots[0], root0)
	require.Equal(t, roots[1], validRootsBeforeFork[0])
	require.Equal(t, roots[2], validRootsBeforeFork[1])
	require.Equal(t, roots[3], validRootsBeforeFork[2])
	require.Len(t, rootTracker.Buffer(), 0)

	// Adding multiple events for same block
	events = []*contracts.RLNMemberRegistered{}

	err = gm.handler(events, 0)
	require.NoError(t, err)

	roots = gm.rootTracker.Roots()
	require.Len(t, roots, 5)

}
