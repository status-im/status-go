package alchemy_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/activityfetcher"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func TestSaveActivity(t *testing.T) {
	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	defer walletDB.Close()

	persistence := activityfetcher.NewPersistence(walletDB)

	sender1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	sender2 := common.HexToAddress("0x0000000000000000000000000000000000000004")
	recipient1 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	recipient2 := common.HexToAddress("0x0000000000000000000000000000000000000005")

	activity := thirdparty.ActivityEntryContainer{
		NextCursor:     "next",
		PreviousCursor: "previous",
		Provider:       "provider",
		Items: []thirdparty.ActivityEntry{
			{
				Timestamp:    1716153600,
				ActivityType: ac.SendAT,
				AmountOut:    (*hexutil.Big)(big.NewInt(1000000000000000000)), // 1 ETH
				TokenOut: &ac.Token{
					TokenType: ac.Native,
					ChainID:   1,
					Address:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
				},
				Sender:          sender1,
				Recipient:       &recipient1,
				ChainIDOut:      ptr(uint64(1)),
				ContractAddress: ptr(common.HexToAddress("0x0000000000000000000000000000000000000003")),
				TxHash:          common.HexToHash("0x123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0"),
				BlockNumber:     (*hexutil.Big)(big.NewInt(17000000)),
			},
			{
				Timestamp:    1716153500,
				ActivityType: ac.SwapAT,
				AmountOut:    (*hexutil.Big)(big.NewInt(1000000000000000000)), // 1 Token A
				AmountIn:     (*hexutil.Big)(big.NewInt(2000000000000000000)), // 2 Token B
				TokenOut: &ac.Token{
					TokenType: ac.Erc20,
					ChainID:   1,
					Address:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
				},
				TokenIn: &ac.Token{
					TokenType: ac.Erc20,
					ChainID:   1,
					Address:   common.HexToAddress("0x2222222222222222222222222222222222222222"),
				},
				Sender:          sender2,
				Recipient:       &recipient2,
				ChainIDOut:      ptr(uint64(1)),
				ChainIDIn:       ptr(uint64(1)),
				ContractAddress: ptr(common.HexToAddress("0x0000000000000000000000000000000000000006")),
				TxHash:          common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
				BlockNumber:     (*hexutil.Big)(big.NewInt(17000001)),
			},
		},
	}

	parameters := thirdparty.ActivityFetchParameters{
		FromBlock: ptr(rpc.BlockNumber(17000000)),
		ToBlock:   ptr(rpc.BlockNumber(17000001)),
		Address:   sender1,
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Both,
	}

	// Save the activity
	err = persistence.SaveActivity(context.Background(), 1, parameters, activity)
	require.NoError(t, err)

	// Test GetLastFetchedBlock
	lastFetchedBlock, lastFetchedTimestamp, err := persistence.GetLastFetchedBlockAndTimestamp(context.Background(), 1, sender1)
	require.NoError(t, err)
	require.NotNil(t, lastFetchedBlock)
	require.Equal(t, rpc.BlockNumber(17000001), *lastFetchedBlock)
	require.NotNil(t, lastFetchedTimestamp)

	// Test GetLastFetchedBlock with no activities
	lastFetchedBlock, lastFetchedTimestamp, err = persistence.GetLastFetchedBlockAndTimestamp(context.Background(), 1, common.HexToAddress("0x0000000000000000000000000000000000000007"))
	require.NoError(t, err)
	require.Nil(t, lastFetchedBlock)
	require.Nil(t, lastFetchedTimestamp)

	// Test retrieving activities for specific addresses and chain IDs
	testCases := []struct {
		name      string
		chainIDs  []uint64
		addresses []common.Address
		limit     uint64
		expected  int // expected number of activities
	}{
		{
			name:      "fetch by sender1",
			chainIDs:  []uint64{1},
			addresses: []common.Address{sender1},
			limit:     10,
			expected:  1,
		},
		{
			name:      "fetch by recipient2",
			chainIDs:  []uint64{1},
			addresses: []common.Address{recipient2},
			limit:     10,
			expected:  1,
		},
		{
			name:      "fetch by multiple addresses",
			chainIDs:  []uint64{1},
			addresses: []common.Address{sender1, sender2, recipient1, recipient2},
			limit:     10,
			expected:  2,
		},
		{
			name:      "fetch with wrong chain ID",
			chainIDs:  []uint64{2},
			addresses: []common.Address{sender1},
			limit:     10,
			expected:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			activities, err := persistence.GetActivity(context.Background(), tc.chainIDs, tc.addresses, tc.limit)
			require.NoError(t, err)
			require.Len(t, activities, tc.expected)

			if tc.expected > 0 {
				// Verify the content of retrieved activities
				for _, act := range activities {
					require.NotZero(t, act.Timestamp)
					require.NotEmpty(t, act.TxHash)
					require.NotNil(t, act.BlockNumber)
				}
			}
		})
	}
}
