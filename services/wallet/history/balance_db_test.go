package history

import (
	"database/sql"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/walletdatabase"
)

func setupBalanceDBTest(t *testing.T) (*BalanceDB, func()) {
	db, err := walletdatabase.InitializeDB(dbsetup.InMemoryPath, "wallet-history-balance_db-tests", 1)
	require.NoError(t, err)
	return NewBalanceDB(db), func() {
		require.NoError(t, db.Close())
	}
}

// generateTestDataForElementCount generates dummy consecutive blocks of data for the same chain_id, address and currency
func generateTestDataForElementCount(count int) (result []*entry) {
	baseDataPoint := entry{
		chainID:     777,
		address:     common.Address{7},
		tokenSymbol: "ETH",
		block:       big.NewInt(11),
		balance:     big.NewInt(101),
		timestamp:   11,
	}

	result = make([]*entry, 0, count)
	for i := 0; i < count; i++ {
		newDataPoint := baseDataPoint
		newDataPoint.block = new(big.Int).Add(baseDataPoint.block, big.NewInt(int64(i)))
		newDataPoint.balance = new(big.Int).Add(baseDataPoint.balance, big.NewInt(int64(i)))
		newDataPoint.timestamp += int64(i)
		result = append(result, &newDataPoint)
	}
	return result
}

func TestBalanceDBAddDataPoint(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoint := generateTestDataForElementCount(1)[0]

	err := bDB.add(testDataPoint, filterWeekly)
	require.NoError(t, err)

	outDataPoint := entry{
		chainID: 0,
		block:   big.NewInt(0),
		balance: big.NewInt(0),
	}
	rows, err := bDB.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok := rows.Next()
	require.True(t, ok)

	bitset := 0
	err = rows.Scan(&outDataPoint.chainID, &outDataPoint.address, &outDataPoint.tokenSymbol, (*bigint.SQLBigInt)(outDataPoint.block), &outDataPoint.timestamp, &bitset, (*bigint.SQLBigIntBytes)(outDataPoint.balance))
	require.NoError(t, err)
	require.NotEqual(t, err, sql.ErrNoRows)
	require.Equal(t, testDataPoint, &outDataPoint)

	ok = rows.Next()
	require.False(t, ok)
}

func TestBalanceDBGetOldestDataPoint(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := len(testDataPoints) - 1; i >= 0; i-- {
		err := bDB.add(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	outDataPoints, _, err := bDB.get(&assetIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].tokenSymbol}, nil, 1, asc)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, outDataPoints[0], testDataPoints[0])
}

func TestBalanceDBGetLatestDataPoint(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bDB.add(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	outDataPoints, _, err := bDB.get(&assetIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].tokenSymbol}, nil, 1, desc)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, outDataPoints[0], testDataPoints[len(testDataPoints)-1])
}

func TestBalanceDBGetFirst(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bDB.add(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	duplicateIndex := 2
	newDataPoint := entry{
		chainID:     testDataPoints[duplicateIndex].chainID,
		address:     common.Address{77},
		tokenSymbol: testDataPoints[duplicateIndex].tokenSymbol,
		block:       new(big.Int).Set(testDataPoints[duplicateIndex].block),
		balance:     big.NewInt(102),
		timestamp:   testDataPoints[duplicateIndex].timestamp,
	}
	err := bDB.add(&newDataPoint, 2)
	require.NoError(t, err)

	outDataPoint, _, err := bDB.getFirst(testDataPoints[duplicateIndex].chainID, testDataPoints[duplicateIndex].block)
	require.NoError(t, err)
	require.NotEqual(t, nil, outDataPoint)
	require.Equal(t, testDataPoints[duplicateIndex], outDataPoint)
}

func TestBalanceDBGetLastEntryForChain(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bDB.add(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	// Same data with different addresses
	for i := 0; i < len(testDataPoints); i++ {
		newDataPoint := testDataPoints[i]
		newDataPoint.address = common.Address{77}
		err := bDB.add(newDataPoint, 1)
		require.NoError(t, err)
	}

	outDataPoint, _, err := bDB.getLastEntryForChain(testDataPoints[0].chainID)
	require.NoError(t, err)
	require.NotEqual(t, nil, outDataPoint)

	expectedDataPoint := testDataPoints[len(testDataPoints)-1]
	require.Equal(t, expectedDataPoint.chainID, outDataPoint.chainID)
	require.Equal(t, expectedDataPoint.tokenSymbol, outDataPoint.tokenSymbol)
	require.Equal(t, expectedDataPoint.block, outDataPoint.block)
	require.Equal(t, expectedDataPoint.timestamp, outDataPoint.timestamp)
	require.Equal(t, expectedDataPoint.balance, outDataPoint.balance)
}

func TestBalanceDBGetDataPointsInTimeRange(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bDB.add(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	startIndex := 1
	endIndex := 3
	outDataPoints, _, err := bDB.filter(&assetIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].tokenSymbol}, nil, &balanceFilter{testDataPoints[startIndex].timestamp, testDataPoints[endIndex].timestamp, 1}, 100, asc)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, len(outDataPoints), endIndex-startIndex+1)
	for i := startIndex; i <= endIndex; i++ {
		require.Equal(t, outDataPoints[i-startIndex], testDataPoints[i])
	}
}

func TestBalanceDBGetClosestDataPointToTimestamp(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bDB.add(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	itemToGetIndex := 2
	outDataPoints, _, err := bDB.filter(&assetIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].tokenSymbol}, nil, &balanceFilter{testDataPoints[itemToGetIndex].timestamp, maxAllRangeTimestamp, 1}, 1, asc)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, len(outDataPoints), 1)
	require.Equal(t, outDataPoints[0], testDataPoints[itemToGetIndex])
}

func TestBalanceDBUpdateUpdateBitset(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(1)

	err := bDB.add(testDataPoints[0], 1)
	require.NoError(t, err)
	err = bDB.add(testDataPoints[0], 2)
	require.Error(t, err, "Expected \"UNIQUE constraint failed: ...\"")
	err = bDB.updateBitset(&assetIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].tokenSymbol}, testDataPoints[0].block, 2)
	require.NoError(t, err)

	outDataPoint := entry{
		chainID: 0,
		block:   big.NewInt(0),
		balance: big.NewInt(0),
	}
	rows, err := bDB.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok := rows.Next()
	require.True(t, ok)

	bitset := 0
	err = rows.Scan(&outDataPoint.chainID, &outDataPoint.address, &outDataPoint.tokenSymbol, (*bigint.SQLBigInt)(outDataPoint.block), &outDataPoint.timestamp, &bitset, (*bigint.SQLBigIntBytes)(outDataPoint.balance))
	require.NoError(t, err)
	require.NotEqual(t, err, sql.ErrNoRows)
	require.Equal(t, testDataPoints[0], &outDataPoint)
	require.Equal(t, 2, bitset)

	ok = rows.Next()
	require.False(t, ok)
}

func TestBalanceDBCheckMissingDataPoint(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	testDataPoint := generateTestDataForElementCount(1)[0]

	err := bDB.add(testDataPoint, 1)
	require.NoError(t, err)

	missingDataPoint := testDataPoint
	missingDataPoint.block = big.NewInt(12)

	outDataPoints, bitset, err := bDB.get(&assetIdentity{missingDataPoint.chainID, missingDataPoint.address, missingDataPoint.tokenSymbol}, missingDataPoint.block, 1, asc)
	require.NoError(t, err)
	require.Equal(t, 0, len(outDataPoints))
	require.Equal(t, 0, len(bitset))
}

func TestBalanceDBBitsetFilter(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	data := generateTestDataForElementCount(3)

	for i := 0; i < len(data); i++ {
		err := bDB.add(data[i], 1<<i)
		require.NoError(t, err)
	}

	for i := 0; i < len(data); i++ {
		outDataPoints, bitset, err := bDB.filter(&assetIdentity{data[0].chainID, data[0].address, data[0].tokenSymbol}, nil, &balanceFilter{
			minTimestamp: minAllRangeTimestamp,
			maxTimestamp: maxAllRangeTimestamp,
			bitset:       expandFlag(1 << i),
		}, 10, asc)
		require.NoError(t, err)
		require.Equal(t, i+1, len(outDataPoints))
		require.Equal(t, bitsetFilter(1<<i), bitset[i])
	}
}

func TestBalanceDBBDataPointUniquenessConstraint(t *testing.T) {
	bDB, cleanDB := setupBalanceDBTest(t)
	defer cleanDB()

	dataPoint := generateTestDataForElementCount(1)[0]

	err := bDB.add(dataPoint, 1)
	require.NoError(t, err)

	testDataPointSame := dataPoint
	testDataPointSame.balance = big.NewInt(102)
	testDataPointSame.timestamp = 12

	err = bDB.add(testDataPointSame, 1)
	require.ErrorContains(t, err, "UNIQUE constraint failed", "should fail because of uniqueness constraint")

	rows, err := bDB.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok := rows.Next()
	require.True(t, ok)
	ok = rows.Next()
	require.False(t, ok)

	testDataPointNew := testDataPointSame
	testDataPointNew.block = big.NewInt(21)

	err = bDB.add(testDataPointNew, 277)
	require.NoError(t, err)

	rows, err = bDB.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok = rows.Next()
	require.True(t, ok)
	ok = rows.Next()
	require.True(t, ok)
	ok = rows.Next()
	require.False(t, ok)

	outDataPoints, bitsets, err := bDB.get(&assetIdentity{testDataPointNew.chainID, testDataPointNew.address, testDataPointNew.tokenSymbol}, testDataPointNew.block, 10, asc)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, 1, len(outDataPoints))
	require.Equal(t, 1, len(bitsets))
	require.Equal(t, testDataPointNew, outDataPoints[0])
	require.Equal(t, bitsetFilter(277), bitsets[0])
}
