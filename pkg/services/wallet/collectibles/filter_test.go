package collectibles

import (
	"context"
	"database/sql"
	"math/big"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/protocol/communities/token"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	"github.com/status-im/status-go/pkg/services/wallet/collectibles/ownership"
	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

func setupTestFilterDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestFilterOwnedCollectibles(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	oDB := ownership.NewOwnershipDB(db)
	cDB := NewCollectibleDataDB(db)

	const nData = 50
	data := thirdparty.GenerateTestCollectiblesData(nData)
	communityData := thirdparty.GenerateTestCollectiblesCommunityData(nData)

	ownerAddresses := []common.Address{
		common.HexToAddress("0x1234"),
		common.HexToAddress("0x5678"),
		common.HexToAddress("0xABCD"),
	}
	randomAddress := common.HexToAddress("0xFFFF")

	dataPerID := make(map[string]thirdparty.CollectibleData)
	communityDataPerID := make(map[string]thirdparty.CollectibleCommunityInfo)
	balancesPerChainIDAndOwner := make(map[w_common.ChainID]map[common.Address][]thirdparty.CollectibleIDBalance)

	var err error

	var commonID thirdparty.CollectibleUniqueID

	for i := 0; i < nData; i++ {
		iData := data[i]
		iCommunityData := communityData[i]

		if i == 1 {
			// Insert a duplicate ID to represent 2 owners having the same ERC1155 collectible
			iData = data[0]
			iCommunityData = communityData[0]
			commonID = iData.ID
		}

		dataPerID[iData.ID.HashKey()] = iData
		communityDataPerID[iData.ID.HashKey()] = iCommunityData

		chainID := iData.ID.ContractID.ChainID
		ownerAddress := ownerAddresses[i%len(ownerAddresses)]

		if _, ok := balancesPerChainIDAndOwner[chainID]; !ok {
			balancesPerChainIDAndOwner[chainID] = make(map[common.Address][]thirdparty.CollectibleIDBalance)
		}
		if _, ok := balancesPerChainIDAndOwner[chainID][ownerAddress]; !ok {
			balancesPerChainIDAndOwner[chainID][ownerAddress] = make([]thirdparty.CollectibleIDBalance, 0, len(data))
		}
		balance := thirdparty.CollectibleIDBalance{
			ID:          iData.ID,
			Balance:     &bigint.BigInt{Int: big.NewInt(int64(i % 10))},
			TxTimestamp: int64(i),
		}
		balancesPerChainIDAndOwner[chainID][ownerAddress] = append(balancesPerChainIDAndOwner[chainID][ownerAddress], balance)
	}

	timestamp := int64(1234567890)

	for chainID, balancesPerOwner := range balancesPerChainIDAndOwner {
		for ownerAddress, balances := range balancesPerOwner {
			_, _, _, err = oDB.Update(chainID, ownerAddress, balances, timestamp)
			require.NoError(t, err)
		}
	}

	err = cDB.SetData(data, true)
	require.NoError(t, err)
	for i := 0; i < nData; i++ {
		err = cDB.SetCommunityInfo(data[i].ID, communityData[i])
		require.NoError(t, err)
	}

	var filter Filter
	var filterIDs []thirdparty.CollectibleUniqueID
	var expectedIDs []thirdparty.CollectibleUniqueID
	var tmpIDs []thirdparty.CollectibleUniqueID

	ctx := context.Background()

	filterChains := []w_common.ChainID{w_common.ChainID(1), w_common.ChainID(2)}
	filterAddresses := []common.Address{ownerAddresses[0], ownerAddresses[1], ownerAddresses[2], randomAddress}

	// Test common case
	filter = allFilter()

	tmpIDs, err = oDB.GetOwnedCollectibles(filterChains, filterAddresses, 0, nData)
	require.NoError(t, err)

	expectedIDs = tmpIDs

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)

	// Test only non-community
	filter = allFilter()
	filter.FilterCommunity = OnlyNonCommunity

	tmpIDs, err = oDB.GetOwnedCollectibles(filterChains, filterAddresses, 0, nData)
	require.NoError(t, err)

	expectedIDs = nil
	for _, id := range tmpIDs {
		if dataPerID[id.HashKey()].CommunityID == "" {
			expectedIDs = append(expectedIDs, id)
		}
	}

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)

	// Test only community
	filter = allFilter()
	filter.FilterCommunity = OnlyCommunity

	tmpIDs, err = oDB.GetOwnedCollectibles(filterChains, filterAddresses, 0, nData)
	require.NoError(t, err)

	expectedIDs = nil
	for _, id := range tmpIDs {
		if dataPerID[id.HashKey()].CommunityID != "" {
			expectedIDs = append(expectedIDs, id)
		}
	}

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)

	// Test specific community
	communityIDa := data[0].CommunityID
	communityIDb := data[1].CommunityID
	communityIDs := []string{communityIDa, communityIDb}

	filter = allFilter()
	filter.CommunityIDs = communityIDs

	tmpIDs, err = oDB.GetOwnedCollectibles(filterChains, filterAddresses, 0, nData)
	require.NoError(t, err)

	expectedIDs = nil
	for _, id := range tmpIDs {
		if dataPerID[id.HashKey()].CommunityID == communityIDa || dataPerID[id.HashKey()].CommunityID == communityIDb {
			expectedIDs = append(expectedIDs, id)
		}
	}

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)

	// Test specific privileges level
	privilegeLevel := token.PrivilegesLevel(2)

	filter = allFilter()
	filter.CommunityPrivilegesLevels = []token.PrivilegesLevel{privilegeLevel}

	tmpIDs, err = oDB.GetOwnedCollectibles(filterChains, filterAddresses, 0, nData)
	require.NoError(t, err)

	expectedIDs = nil
	for _, id := range tmpIDs {
		if communityDataPerID[id.HashKey()].PrivilegesLevel == privilegeLevel {
			expectedIDs = append(expectedIDs, id)
		}
	}

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)

	// Test specific collectible IDs
	tmpIDs, err = oDB.GetOwnedCollectibles(filterChains, filterAddresses, 0, nData)
	require.NoError(t, err)

	filter = allFilter()
	for i := 0; i < 5; i++ {
		filter.CollectibleIDs = append(filter.CollectibleIDs, tmpIDs[i*2])
	}
	expectedIDs = filter.CollectibleIDs

	filter.CollectibleIDs = append(filter.CollectibleIDs, thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: w_common.ChainID(1),
			Address: common.HexToAddress("0x1234"),
		},
		TokenID: &bigint.BigInt{Int: big.NewInt(9999999)},
	})

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)

	// Test collectible ID owned by both accounts 0 and 1
	filterChains = []w_common.ChainID{commonID.ContractID.ChainID}
	filterAddresses = []common.Address{ownerAddresses[0], ownerAddresses[1]}

	filter = allFilter()
	filter.CollectibleIDs = append(filter.CollectibleIDs, commonID)
	expectedIDs = filter.CollectibleIDs

	filterIDs, err = filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, 0, nData)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, filterIDs)
}

// TestFilterOwnedCollectiblesPagingIsDeterministic pins the pagination contract of
// filterOwnedCollectibles: consecutive LIMIT/OFFSET pages must not overlap or skip rows,
// and the result must be sorted by (chain_id, contract_address, token_id), regardless of
// the order rows were inserted/stored in, or of which owner address happens to hold which
// contract.
func TestFilterOwnedCollectiblesPagingIsDeterministic(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	oDB := ownership.NewOwnershipDB(db)

	// Two owners, chosen so ownerLexLow < ownerLexHigh as raw address bytes: the
	// covering index on (chain_id, owner_address, contract_address, token_id) makes
	// SQLite naturally group rows by owner within a chain. Giving the
	// lexicographically-low owner the high contract address (and vice versa) makes
	// that per-owner grouping disagree with the required (chain, contract, token) order.
	ownerLexLow := common.HexToAddress("0x1111")
	ownerLexHigh := common.HexToAddress("0x9999")

	chainLow := w_common.ChainID(10)
	chainHigh := w_common.ChainID(20)

	contractLow := common.HexToAddress("0xAAAA")
	contractMid := common.HexToAddress("0x5555")
	contractHigh := common.HexToAddress("0xEEEE")

	newBalance := func(chainID w_common.ChainID, contract common.Address, tokenID int64) thirdparty.CollectibleIDBalance {
		return thirdparty.CollectibleIDBalance{
			ID: thirdparty.CollectibleUniqueID{
				ContractID: thirdparty.ContractID{ChainID: chainID, Address: contract},
				TokenID:    &bigint.BigInt{Int: big.NewInt(tokenID)},
			},
			Balance:     &bigint.BigInt{Int: big.NewInt(1)},
			TxTimestamp: 1,
		}
	}

	// chain low: the lex-low owner holds the HIGH contract, the lex-high owner holds the
	// LOW contract, so per-owner index grouping and per-contract sort order disagree.
	lexLowOnChainLow := []thirdparty.CollectibleIDBalance{
		newBalance(chainLow, contractHigh, 1),
		newBalance(chainLow, contractHigh, 2),
	}
	lexHighOnChainLow := []thirdparty.CollectibleIDBalance{
		newBalance(chainLow, contractLow, 1),
		newBalance(chainLow, contractLow, 2),
	}
	// chain high: also split across owners/contracts, unscrambled relative to the
	// chain-low case, so this test doesn't rely on a single lucky arrangement.
	lexLowOnChainHigh := []thirdparty.CollectibleIDBalance{
		newBalance(chainHigh, contractLow, 5),
	}
	lexHighOnChainHigh := []thirdparty.CollectibleIDBalance{
		newBalance(chainHigh, contractMid, 1),
	}

	_, _, _, err := oDB.Update(chainLow, ownerLexLow, lexLowOnChainLow, 1)
	require.NoError(t, err)
	_, _, _, err = oDB.Update(chainLow, ownerLexHigh, lexHighOnChainLow, 2)
	require.NoError(t, err)
	_, _, _, err = oDB.Update(chainHigh, ownerLexLow, lexLowOnChainHigh, 3)
	require.NoError(t, err)
	_, _, _, err = oDB.Update(chainHigh, ownerLexHigh, lexHighOnChainHigh, 4)
	require.NoError(t, err)

	allBalances := append(append(append(append([]thirdparty.CollectibleIDBalance{},
		lexLowOnChainLow...), lexHighOnChainLow...), lexLowOnChainHigh...), lexHighOnChainHigh...)
	expectedIDs := make([]thirdparty.CollectibleUniqueID, 0, len(allBalances))
	for _, b := range allBalances {
		expectedIDs = append(expectedIDs, b.ID)
	}
	sort.Slice(expectedIDs, func(i, j int) bool {
		a, b := expectedIDs[i], expectedIDs[j]
		if a.ContractID.ChainID != b.ContractID.ChainID {
			return a.ContractID.ChainID < b.ContractID.ChainID
		}
		addrCmp := bytesCompare(a.ContractID.Address.Bytes(), b.ContractID.Address.Bytes())
		if addrCmp != 0 {
			return addrCmp < 0
		}
		return a.TokenID.Cmp(b.TokenID.Int) < 0
	})

	require.Len(t, expectedIDs, 6)

	ctx := context.Background()
	filter := allFilter()
	filterChains := []w_common.ChainID{chainLow, chainHigh}
	filterAddresses := []common.Address{ownerLexLow, ownerLexHigh}

	const pageSize = 2
	var pages [][]thirdparty.CollectibleUniqueID
	var gotIDs []thirdparty.CollectibleUniqueID
	for offset := 0; offset < len(expectedIDs); offset += pageSize {
		page, err := filterOwnedCollectibles(ctx, db, filterChains, filterAddresses, filter, offset, pageSize)
		require.NoError(t, err)
		pages = append(pages, page)
		gotIDs = append(gotIDs, page...)
	}

	// (a) concatenated pages equal the full set, no duplicates, no misses.
	require.Equal(t, expectedIDs, gotIDs, "pages must concatenate to the full, non-overlapping set")

	// (b) the returned order is sorted by (chain_id, contract_address, token_id). This is
	// the assertion that pins the contract even if SQLite happens to return sorted rows
	// for other reasons (e.g. a DISTINCT implementation that sorts internally).
	require.True(t, sort.SliceIsSorted(gotIDs, func(i, j int) bool {
		a, b := gotIDs[i], gotIDs[j]
		if a.ContractID.ChainID != b.ContractID.ChainID {
			return a.ContractID.ChainID < b.ContractID.ChainID
		}
		addrCmp := bytesCompare(a.ContractID.Address.Bytes(), b.ContractID.Address.Bytes())
		if addrCmp != 0 {
			return addrCmp < 0
		}
		return a.TokenID.Cmp(b.TokenID.Int) < 0
	}), "result must be ordered by (chain_id, contract_address, token_id)")
}

func bytesCompare(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return len(a) - len(b)
}
