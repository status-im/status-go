package alchemy_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/activity/alchemy"
)

var (
	fixtureAddress = common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e")
	otherAddress   = common.HexToAddress("0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7")

	defaultCategories = []alchemy.TransferCategory{alchemy.TransferCategoryExternal, alchemy.TransferCategoryErc20}
	allCategories     = []alchemy.TransferCategory{
		alchemy.TransferCategoryExternal, alchemy.TransferCategoryErc20,
		alchemy.TransferCategoryErc721, alchemy.TransferCategoryErc1155, alchemy.TransferCategorySpecialNft,
	}
)

const batchErc1155Transfer = `{
	"blockNum": "0x6a8000",
	"uniqueId": "0x1111111111111111111111111111111111111111111111111111111111111111:log:1",
	"hash": "0x1111111111111111111111111111111111111111111111111111111111111111",
	"from": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
	"to": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
	"erc1155Metadata": [{"tokenId": "0x0a", "value": "0x02"}, {"tokenId": "0x0b", "value": "0x05"}],
	"tokenId": null,
	"asset": "FA1155",
	"category": "erc1155",
	"rawContract": {"value": null, "address": "0x1ed60fedff775d500dde21a974cd4e92e0047cc8", "decimal": null},
	"metadata": {"blockTimestamp": "2024-11-01T10:00:00.000Z"}
}`

func fixtureTransfers(t *testing.T) []alchemy.Transfer {
	var response alchemy.GetAssetTranfersResponse
	require.NoError(t, json.Unmarshal([]byte(getAssetTransfersResponseData2), &response))
	return response.Transfers
}

func setupPersistence(t *testing.T) *alchemy.Persistence {
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return alchemy.NewPersistence(db)
}

func countCategories(transfers []alchemy.Transfer, categories ...alchemy.TransferCategory) int {
	count := 0
	for _, transfer := range transfers {
		for _, c := range categories {
			if transfer.Category == c {
				count++
			}
		}
	}
	return count
}

func selfTransfers(transfers []alchemy.Transfer) []alchemy.Transfer {
	var result []alchemy.Transfer
	for _, transfer := range transfers {
		if transfer.ToAddress != nil && *transfer.ToAddress == transfer.FromAddress {
			result = append(result, transfer)
		}
	}
	return result
}

func TestGetHistoryFromFixtures(t *testing.T) {
	ctx := context.Background()
	p := setupPersistence(t)
	transfers := fixtureTransfers(t)
	self := selfTransfers(transfers)
	require.Len(t, self, 2)

	// The fetcher stores a self transfer twice: from the outgoing and from the incoming query.
	require.NoError(t, p.SaveTransfers(transfers, 1, fixtureAddress))
	require.NoError(t, p.SaveTransfers(self, 1, fixtureAddress))

	history, err := p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress}, defaultCategories, 1000, 0)
	require.NoError(t, err)
	require.Len(t, history, countCategories(transfers, defaultCategories...))

	seen := map[string]bool{}
	for i, entry := range history {
		require.False(t, seen[entry.Transfer.UniqueID], "duplicate %s", entry.Transfer.UniqueID)
		seen[entry.Transfer.UniqueID] = true
		require.Contains(t, defaultCategories, entry.Transfer.Category)
		if i > 0 {
			require.False(t, entry.Transfer.Metadata.BlockTimestamp.After(history[i-1].Transfer.Metadata.BlockTimestamp), "not newest first")
		}
	}

	withCollectibles, err := p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress}, allCategories, 1000, 0)
	require.NoError(t, err)
	require.Len(t, withCollectibles, len(transfers))

	none, err := p.GetHistory(ctx, []uint64{10}, []common.Address{fixtureAddress}, allCategories, 1000, 0)
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestGetHistoryPagination(t *testing.T) {
	ctx := context.Background()
	p := setupPersistence(t)
	require.NoError(t, p.SaveTransfers(fixtureTransfers(t), 1, fixtureAddress))

	all, err := p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress}, allCategories, 1000, 0)
	require.NoError(t, err)

	var paged []alchemy.StoredTransfer
	for offset := 0; offset < len(all); offset += 10 {
		page, err := p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress}, allCategories, 10, offset)
		require.NoError(t, err)
		paged = append(paged, page...)
	}
	require.Equal(t, all, paged)
}

func TestGetHistoryBatchErc1155(t *testing.T) {
	ctx := context.Background()
	p := setupPersistence(t)
	var transfer alchemy.Transfer
	require.NoError(t, json.Unmarshal([]byte(batchErc1155Transfer), &transfer))
	require.NoError(t, p.SaveTransfers([]alchemy.Transfer{transfer}, 1, fixtureAddress))

	history, err := p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress}, allCategories, 1000, 0)
	require.NoError(t, err)
	require.Len(t, history, 2)

	first := alchemy.ToHistoryTransfer(history[0])
	second := alchemy.ToHistoryTransfer(history[1])
	require.Equal(t, "10", *first.TokenID)
	require.Equal(t, "2", first.Amount)
	require.Equal(t, "11", *second.TokenID)
	require.Equal(t, "5", second.Amount)
	require.Equal(t, alchemy.HistoryDirectionIn, first.Direction)
	require.Equal(t, uint64(0), *first.Decimals)

	history, err = p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress}, defaultCategories, 1000, 0)
	require.NoError(t, err)
	require.Empty(t, history)
}

func TestGetHistoryBetweenOwnAddresses(t *testing.T) {
	ctx := context.Background()
	p := setupPersistence(t)
	var erc20 alchemy.Transfer
	for _, transfer := range fixtureTransfers(t) {
		if transfer.Category == alchemy.TransferCategoryErc20 && transfer.ToAddress != nil && *transfer.ToAddress == otherAddress {
			erc20 = transfer
			break
		}
	}
	require.NotEmpty(t, erc20.UniqueID)

	// Each side's fetcher stores the transfer under its own address.
	require.NoError(t, p.SaveTransfers([]alchemy.Transfer{erc20}, 1, fixtureAddress))
	require.NoError(t, p.SaveTransfers([]alchemy.Transfer{erc20}, 1, otherAddress))

	history, err := p.GetHistory(ctx, []uint64{1}, []common.Address{fixtureAddress, otherAddress}, defaultCategories, 1000, 0)
	require.NoError(t, err)
	require.Len(t, history, 2)

	byAddress := map[common.Address]alchemy.HistoryTransfer{}
	for _, entry := range history {
		byAddress[entry.Address] = alchemy.ToHistoryTransfer(entry)
	}
	out, in := byAddress[fixtureAddress], byAddress[otherAddress]
	require.Equal(t, alchemy.HistoryDirectionOut, out.Direction)
	require.Equal(t, alchemy.HistoryDirectionIn, in.Direction)
	require.Equal(t, "-"+in.Amount, out.Amount)
	require.Equal(t, *erc20.RawContract.Address, out.TokenAddress)
}

func TestToHistoryTransfer(t *testing.T) {
	byID := map[string]alchemy.Transfer{}
	for _, transfer := range fixtureTransfers(t) {
		byID[transfer.UniqueID] = transfer
	}
	convert := func(uniqueID string) alchemy.HistoryTransfer {
		transfer, ok := byID[uniqueID]
		require.True(t, ok, uniqueID)
		return alchemy.ToHistoryTransfer(alchemy.StoredTransfer{ChainID: 1, Address: fixtureAddress, Transfer: transfer})
	}

	self := convert("0xe3fd213c8794cb929b97f2ad171b5c55dde1965ce8ac52114f83fe33db86750c:external")
	require.Equal(t, alchemy.HistoryDirectionSelf, self.Direction)
	require.Equal(t, common.Address{}, self.TokenAddress)
	require.Equal(t, "10000000000000000", self.Amount)
	require.Equal(t, "0.01", *self.Value)
	require.Equal(t, uint64(18), *self.Decimals)
	require.Equal(t, "ETH", self.Symbol)
	require.Equal(t, uint64(0x5c0e09), self.BlockNumber)
	require.Equal(t, int64(1717437528), self.Timestamp)
	require.Nil(t, self.TokenID)

	nft := convert("0x0e0c4834de556d22a03c1727435f88fb38cefd7c67ee0ab98c6aa1ccfb19a7cb:log:66")
	require.Equal(t, alchemy.HistoryDirectionOut, nft.Direction)
	require.Equal(t, "51", *nft.TokenID)
	require.Equal(t, "-1", nft.Amount)
	require.Equal(t, "-1", *nft.Value)
	require.Equal(t, common.HexToAddress("0x9f64932be34d5d897c4253d17707b50921f372b6"), nft.TokenAddress)
}

func TestToHistoryTransferValue(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		decimal string
		amount  string
		value   *string
	}{
		{"whole", "0xde0b6b3a7640000", `"0x12"`, "-1000000000000000000", strPtr("-1")},
		{"fraction", "0x1e240", `"0x6"`, "-123456", strPtr("-0.123456")},
		{"tiny", "0x1", `"0x12"`, "-1", strPtr("-0.000000000000000001")},
		{"zero", "0x0", `"0x12"`, "0", strPtr("0")},
		{"unknown decimals", "0x1e240", "null", "-123456", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := `{"blockNum":"0x1","uniqueId":"u","hash":"0x2222222222222222222222222222222222222222222222222222222222222222","from":"` + fixtureAddress.Hex() + `","to":"` + otherAddress.Hex() +
				`","asset":"TKN","category":"erc20","rawContract":{"value":"` + c.raw + `","address":"0x744d70fdbe2ba4cf95131626614a1763df805b9e","decimal":` + c.decimal +
				`},"metadata":{"blockTimestamp":"2024-01-01T00:00:00Z"}}`
			var transfer alchemy.Transfer
			require.NoError(t, json.Unmarshal([]byte(data), &transfer))
			result := alchemy.ToHistoryTransfer(alchemy.StoredTransfer{ChainID: 1, Address: fixtureAddress, Transfer: transfer})
			require.Equal(t, c.amount, result.Amount)
			require.Equal(t, c.value, result.Value)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
