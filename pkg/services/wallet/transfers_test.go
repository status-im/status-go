package wallet

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/activityfetcher"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/activity/alchemy"
)

type fakeTransferFetcher struct {
	supported map[uint64]bool
	complete  map[activityfetcher.FetchKey]bool
	keys      []activityfetcher.FetchKey
	deadline  bool
}

func (f *fakeTransferFetcher) IsChainSupported(chainID uint64) bool {
	return f.supported[chainID]
}

func (f *fakeTransferFetcher) FetchAndWait(ctx context.Context, keys []activityfetcher.FetchKey) map[activityfetcher.FetchKey]bool {
	f.keys = keys
	_, f.deadline = ctx.Deadline()
	return f.complete
}

var (
	transfersOwner = common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e")
	transfersPeer  = common.HexToAddress("0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7")
)

const transfersTestData = `[
	{
		"blockNum": "0x10", "uniqueId": "0xaa:external", "hash": "0x00000000000000000000000000000000000000000000000000000000000000aa",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e", "to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"asset": "ETH", "category": "external",
		"rawContract": {"value": "0xde0b6b3a7640000", "address": null, "decimal": "0x12"},
		"metadata": {"blockTimestamp": "2024-01-02T00:00:00Z"}
	},
	{
		"blockNum": "0x20", "uniqueId": "0xbb:log:1", "hash": "0x00000000000000000000000000000000000000000000000000000000000000bb",
		"from": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7", "to": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"tokenId": "0x33", "asset": "NFT", "category": "erc721",
		"rawContract": {"value": null, "address": "0x9f64932be34d5d897c4253d17707b50921f372b6", "decimal": null},
		"metadata": {"blockTimestamp": "2024-01-03T00:00:00Z"}
	}
]`

func newTestTransferHistory(t *testing.T, fetcher *fakeTransferFetcher) *transferHistory {
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	var transfers []alchemy.Transfer
	require.NoError(t, json.Unmarshal([]byte(transfersTestData), &transfers))
	persistence := alchemy.NewPersistence(db)
	require.NoError(t, persistence.SaveTransfers(transfers, 1, transfersOwner))

	return &transferHistory{fetcher: fetcher, storage: persistence, fetchTimeout: time.Second}
}

func TestGetTransfers(t *testing.T) {
	ownerMainnet := activityfetcher.FetchKey{Address: transfersOwner, ChainID: 1}
	ownerOptimism := activityfetcher.FetchKey{Address: transfersOwner, ChainID: 10}
	fetcher := &fakeTransferFetcher{
		supported: map[uint64]bool{1: true, 10: true},
		complete:  map[activityfetcher.FetchKey]bool{ownerMainnet: true},
	}
	h := newTestTransferHistory(t, fetcher)

	response, err := h.getTransfers(context.Background(), []common.Address{transfersOwner, transfersOwner}, []uint64{1, 10}, false, 100, 0)
	require.NoError(t, err)
	require.ElementsMatch(t, []activityfetcher.FetchKey{ownerMainnet, ownerOptimism}, fetcher.keys)
	require.True(t, fetcher.deadline)

	require.Len(t, response.Transfers, 1)
	require.Equal(t, alchemy.TransferCategoryExternal, response.Transfers[0].Category)
	require.Equal(t, "-1000000000000000000", response.Transfers[0].Amount)

	require.Len(t, response.Coverage, 2)
	require.Equal(t, TransfersCoverage{Address: transfersOwner, ChainID: 1, Complete: true, LastFetchedBlock: uint64Ptr(0x20)}, response.Coverage[0])
	require.Equal(t, TransfersCoverage{Address: transfersOwner, ChainID: 10, Complete: false}, response.Coverage[1])

	response, err = h.getTransfers(context.Background(), []common.Address{transfersOwner}, []uint64{1}, true, 100, 0)
	require.NoError(t, err)
	require.Len(t, response.Transfers, 2)
	require.Equal(t, alchemy.TransferCategoryErc721, response.Transfers[0].Category)
	require.Equal(t, "51", *response.Transfers[0].TokenID)

	response, err = h.getTransfers(context.Background(), []common.Address{transfersOwner}, []uint64{1}, true, 1, 1)
	require.NoError(t, err)
	require.Len(t, response.Transfers, 1)
	require.Equal(t, alchemy.TransferCategoryExternal, response.Transfers[0].Category)

	response, err = h.getTransfers(context.Background(), []common.Address{transfersPeer}, []uint64{1}, true, 100, 0)
	require.NoError(t, err)
	require.Empty(t, response.Transfers)
	require.NotNil(t, response.Transfers)
}

func TestGetTransfersErrors(t *testing.T) {
	fetcher := &fakeTransferFetcher{supported: map[uint64]bool{1: true}}
	h := newTestTransferHistory(t, fetcher)
	ctx := context.Background()
	owner := []common.Address{transfersOwner}

	_, err := h.getTransfers(ctx, owner, []uint64{1, 56}, false, 100, 0)
	require.EqualError(t, err, "no history provider for chain 56")
	require.Nil(t, fetcher.keys, "must not fetch when a chain has no provider")

	_, err = h.getTransfers(ctx, nil, []uint64{1}, false, 100, 0)
	require.Error(t, err)
	_, err = h.getTransfers(ctx, owner, nil, false, 100, 0)
	require.Error(t, err)
	_, err = h.getTransfers(ctx, owner, []uint64{1}, false, 0, 0)
	require.Error(t, err)
	_, err = h.getTransfers(ctx, owner, []uint64{1}, false, transfersMaxLimit+1, 0)
	require.Error(t, err)
	_, err = h.getTransfers(ctx, owner, []uint64{1}, false, 100, -1)
	require.Error(t, err)
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
