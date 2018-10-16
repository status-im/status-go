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
			Topics: []common.Hash{
				{1},
				{1, 1},
			},
		},
		{
			BlockNumber: 2,
			BlockHash:   common.Hash{2, 2},
			Address:     common.Address{2, 2, 2},
			Topics: []common.Hash{
				{1},
				{2, 2},
			},
		},
	}

	type testCase struct {
		description  string
		crit         ethereum.FilterQuery
		expectedLogs []types.Log
	}

	for _, tc := range []testCase{
		{
			description:  "All",
			crit:         ethereum.FilterQuery{},
			expectedLogs: []types.Log{logs[0], logs[1]},
		},
		{
			description:  "LimitedByBlock",
			crit:         ethereum.FilterQuery{ToBlock: big.NewInt(1)},
			expectedLogs: []types.Log{logs[0]},
		},
		{
			description:  "LimitedByAddress",
			crit:         ethereum.FilterQuery{Addresses: []common.Address{logs[1].Address}},
			expectedLogs: []types.Log{logs[1]},
		},
		{
			description:  "LimitedByAddress",
			crit:         ethereum.FilterQuery{Addresses: []common.Address{logs[1].Address}},
			expectedLogs: []types.Log{logs[1]},
		},
		{
			description: "MoreTopicsThanInLogs",
			crit:        ethereum.FilterQuery{Topics: make([][]common.Hash, 3)},
		},
		{
			description:  "Wildcard",
			crit:         ethereum.FilterQuery{Topics: make([][]common.Hash, 1)},
			expectedLogs: []types.Log{logs[0], logs[1]},
		},
		{
			description:  "LimitedBySecondTopic",
			crit:         ethereum.FilterQuery{Topics: [][]common.Hash{{}, logs[1].Topics}},
			expectedLogs: []types.Log{logs[1]},
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			rst := filterLogs(logs, tc.crit)
			require.Equal(t, tc.expectedLogs, rst)
		})
	}
}

func TestAdjustFromBlock(t *testing.T) {
	type testCase struct {
		description string
		initial     ethereum.FilterQuery
		result      ethereum.FilterQuery
	}

	for _, tc := range []testCase{
		{
			"ToBlockHigherThenLatest",
			ethereum.FilterQuery{ToBlock: big.NewInt(10)},
			ethereum.FilterQuery{ToBlock: big.NewInt(10)},
		},
		{
			"FromBlockIsPending",
			ethereum.FilterQuery{FromBlock: big.NewInt(-2)},
			ethereum.FilterQuery{FromBlock: big.NewInt(-2)},
		},
		{
			"FromBlockIsOlderThenLatest",
			ethereum.FilterQuery{FromBlock: big.NewInt(10)},
			ethereum.FilterQuery{FromBlock: big.NewInt(-1)},
		},
		{
			"NotInterestedInLatestBlocks",
			ethereum.FilterQuery{FromBlock: big.NewInt(10), ToBlock: big.NewInt(15)},
			ethereum.FilterQuery{FromBlock: big.NewInt(10), ToBlock: big.NewInt(15)},
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			adjustFromBlock(&tc.initial)
			require.Equal(t, tc.result, tc.initial)
		})
	}
}
