package rpcfilters

import (
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFilterLogs(t *testing.T) {
	logs := []types.Log{
		{
			BlockNumber: 1,
			BlockHash:   common.Hash{1, 1},
			Address:     common.Address{1, 1, 1},
			Topics:      []common.Hash{common.Hash{1}, common.Hash{1, 1}},
		},
		{
			BlockNumber: 2,
			BlockHash:   common.Hash{2, 2},
			Address:     common.Address{2, 2, 2},
			Topics:      []common.Hash{common.Hash{1}, common.Hash{2, 2}},
		},
	}

	type testCase struct {
		description string

		blockNum  uint64
		blockHash common.Hash
		crit      ethereum.FilterQuery

		expectedLogs  []types.Log
		expectedBlock uint64
		expectedHash  common.Hash
	}

	for _, tc := range []testCase{
		{
			description:   "All",
			crit:          ethereum.FilterQuery{},
			expectedLogs:  []types.Log{logs[0], logs[1]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			description:   "LimitedByBlock",
			crit:          ethereum.FilterQuery{ToBlock: big.NewInt(1)},
			expectedLogs:  []types.Log{logs[0]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			description:   "LimitedByAddress",
			crit:          ethereum.FilterQuery{Addresses: []common.Address{logs[1].Address}},
			expectedLogs:  []types.Log{logs[1]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			description:   "LimitedByAddress",
			crit:          ethereum.FilterQuery{Addresses: []common.Address{logs[1].Address}},
			expectedLogs:  []types.Log{logs[1]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			description:   "MoreTopicsThanInLogs",
			crit:          ethereum.FilterQuery{Topics: make([][]common.Hash, 3)},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			description:   "Wildcard",
			crit:          ethereum.FilterQuery{Topics: make([][]common.Hash, 1)},
			expectedLogs:  []types.Log{logs[0], logs[1]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			description: "LimitedBySecondTopic",
			crit: ethereum.FilterQuery{Topics: [][]common.Hash{
				[]common.Hash{}, logs[1].Topics}},
			expectedLogs:  []types.Log{logs[1]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			blockNum:      logs[1].BlockNumber,
			blockHash:     logs[1].BlockHash,
			description:   "LimitedBySeenBlock",
			crit:          ethereum.FilterQuery{},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
		{
			blockNum:      logs[1].BlockNumber,
			blockHash:     common.Hash{7, 7, 7},
			description:   "SeenBlockDifferenthash",
			crit:          ethereum.FilterQuery{},
			expectedLogs:  []types.Log{logs[1]},
			expectedBlock: logs[1].BlockNumber,
			expectedHash:  logs[1].BlockHash,
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			rst, num, hash := filterLogs(logs, tc.crit, tc.blockNum, tc.blockHash)
			require.Equal(t, tc.expectedLogs, rst)
			require.Equal(t, tc.expectedBlock, num)
			require.Equal(t, tc.expectedHash, hash)
		})
	}

}
