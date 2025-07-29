package alchemy_test

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

var jsonTransfers = `[{
    "asset": "ETH",
    "blockNum": "0x15f3d0f",
    "category": "external",
    "from": "0xe0798e0070d223bed269267bba11f4abffe27a88",
    "hash": "0x90881ea417474e643fbd9c533f10a2277b6e52e99c218bd933baba7929fd1afa",
    "metadata": {
        "blockTimestamp": "2025-07-28T16:14:59Z"
    },
    "rawContract": {
        "address": null,
        "decimal": "0x12",
        "value": "0x38d7ea4c68000"
    },
    "to": "0x0439e60f02a8900a951603950d8d4527f400c3f1",
    "tokenId": null,
    "uniqueId": "0x90881ea417474e643fbd9c533f10a2277b6e52e99c218bd933baba7929fd1afa:external",
    "value": 0.001
}]`

func TestSaveTransfers(t *testing.T) {

	var transfers []alchemy.Transfer

	err := json.Unmarshal([]byte(jsonTransfers), &transfers)
	if err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}
	fmt.Println("transfer:", transfers[0])

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	defer walletDB.Close()

	persistence := alchemy.NewPersistence(walletDB)
	address1 := common.HexToAddress("0xe0798e0070d223bed269267bba11f4abffe27a88")

	// Save the activity
	err = persistence.SaveTransfers(transfers, 1, address1)
	require.NoError(t, err)

	// TBD: Test GetLastFetchedBlock

	// TBD: Test GetLastFetchedBlock with no activities

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
			addresses: []common.Address{address1},
			limit:     10,
			expected:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			retrievedTransfers, err := persistence.GetTransfers(tc.chainIDs, tc.addresses, tc.limit)
			require.NoError(t, err)
			require.Len(t, retrievedTransfers, tc.expected)

			if tc.expected > 0 {
				// Verify the content of retrieved activities
				for _, transfer := range retrievedTransfers {
					require.NotEmpty(t, transfer.Hash)
					require.NotNil(t, transfer.BlockNum)
				}
			}
		})
	}
}
