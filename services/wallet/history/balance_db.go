package history

import (
	"database/sql"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type BalanceDB struct {
	db *sql.DB
}

func NewBalanceDB(sqlDb *sql.DB) *BalanceDB {
	return &BalanceDB{
		db: sqlDb,
	}
}

// entry represents a single row in the balance_history table
type entry struct {
	chainID     uint64
	address     common.Address
	tokenSymbol string
	block       *big.Int
	timestamp   int64
	balance     *big.Int
}

// bitsetFilter stores the time interval for which the data points are matching
type bitsetFilter int

const (
	minAllRangeTimestamp  = 0
	maxAllRangeTimestamp  = math.MaxInt64
	bitsetFilterFlagCount = 30
)

// expandFlag will generate a bitset that matches all lower value flags (fills the less significant bits of the flag with 1; e.g. 0b1000 -> 0b1111)
func expandFlag(flag bitsetFilter) bitsetFilter {
	return (flag << 1) - 1
}

func (b *BalanceDB) add(entry *entry, bitset bitsetFilter) error {
	_, err := b.db.Exec("INSERT INTO balance_history (chain_id, address, currency, block, timestamp, bitset, balance) VALUES (?, ?, ?, ?, ?, ?, ?)", entry.chainID, entry.address, entry.tokenSymbol, (*bigint.SQLBigInt)(entry.block), entry.timestamp, int(bitset), (*bigint.SQLBigIntBytes)(entry.balance))
	return err
}

type sortDirection = int

const (
	asc  sortDirection = 0
	desc sortDirection = 1
)

type assetIdentity struct {
	ChainID     uint64
	Address     common.Address
	TokenSymbol string
}

// bitset is used so higher values can include lower values to simulate time interval levels and high granularity intervals include lower ones
// minTimestamp and maxTimestamp interval filter the results by timestamp.
type balanceFilter struct {
	minTimestamp int64
	maxTimestamp int64
	bitset       bitsetFilter
}

// filters returns a sorted list of entries, empty array if none is found for the given input or nil if error
// if startingAtBlock is provided, the result will start with the provided block number or the next available one
// if startingAtBlock is NOT provided the result will begin from the first available block that matches filter.minTimestamp
// sort defines the order of the result by block number (which correlates also with timestamp)
func (b *BalanceDB) filter(identity *assetIdentity, startingAtBlock *big.Int, filter *balanceFilter, maxEntries int, sort sortDirection) (entries []*entry, bitsetList []bitsetFilter, err error) {
	// Start from the first block in case a specific one was not provided
	if startingAtBlock == nil {
		startingAtBlock = big.NewInt(0)
	}
	// We are interested in order by timestamp, but we request by block number that correlates to the order of timestamp and it is indexed
	var queryStr string
	rawQueryStr := "SELECT block, timestamp, balance, bitset FROM balance_history WHERE chain_id = ? AND address = ? AND currency = ? AND block >= ? AND timestamp BETWEEN ? AND ? AND (bitset & ?) > 0 ORDER BY block %s LIMIT ?"
	if sort == asc {
		queryStr = fmt.Sprintf(rawQueryStr, "ASC")
	} else {
		queryStr = fmt.Sprintf(rawQueryStr, "DESC")
	}
	rows, err := b.db.Query(queryStr, identity.ChainID, identity.Address, identity.TokenSymbol, (*bigint.SQLBigInt)(startingAtBlock), filter.minTimestamp, filter.maxTimestamp, filter.bitset, maxEntries)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	result := make([]*entry, 0)
	for rows.Next() {
		entry := &entry{
			chainID:     0,
			address:     identity.Address,
			tokenSymbol: identity.TokenSymbol,
			block:       new(big.Int),
			balance:     new(big.Int),
		}
		var bitset int
		err := rows.Scan((*bigint.SQLBigInt)(entry.block), &entry.timestamp, (*bigint.SQLBigIntBytes)(entry.balance), &bitset)
		if err != nil {
			return nil, nil, err
		}
		entry.chainID = identity.ChainID
		result = append(result, entry)
		bitsetList = append(bitsetList, bitsetFilter(bitset))
	}
	return result, bitsetList, nil
}

// get calls filter that matches all entries
func (b *BalanceDB) get(identity *assetIdentity, startingAtBlock *big.Int, maxEntries int, sort sortDirection) (entries []*entry, bitsetList []bitsetFilter, err error) {
	return b.filter(identity, startingAtBlock, &balanceFilter{
		minTimestamp: minAllRangeTimestamp,
		maxTimestamp: maxAllRangeTimestamp,
		bitset:       expandFlag(1 << bitsetFilterFlagCount),
	}, maxEntries, sort)
}

// getFirst returns the first entry for the block or nil if no entry is found
func (b *BalanceDB) getFirst(chainID uint64, block *big.Int) (res *entry, bitset bitsetFilter, err error) {
	res = &entry{
		chainID: chainID,
		block:   new(big.Int).Set(block),
		balance: new(big.Int),
	}

	queryStr := "SELECT address, currency, timestamp, balance, bitset FROM balance_history WHERE chain_id = ? AND block = ?"
	row := b.db.QueryRow(queryStr, chainID, (*bigint.SQLBigInt)(block))
	var bitsetRaw int

	err = row.Scan(&res.address, &res.tokenSymbol, &res.timestamp, (*bigint.SQLBigIntBytes)(res.balance), &bitsetRaw)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}

	return res, bitsetFilter(bitsetRaw), nil
}

// getFirst returns the last entry for the chainID or nil if no entry is found
func (b *BalanceDB) getLastEntryForChain(chainID uint64) (res *entry, bitset bitsetFilter, err error) {
	res = &entry{
		chainID: chainID,
		block:   new(big.Int),
		balance: new(big.Int),
	}

	queryStr := "SELECT address, currency, timestamp, block, balance, bitset FROM balance_history WHERE chain_id = ? ORDER BY block DESC"
	row := b.db.QueryRow(queryStr, chainID)
	var bitsetRaw int

	err = row.Scan(&res.address, &res.tokenSymbol, &res.timestamp, (*bigint.SQLBigInt)(res.block), (*bigint.SQLBigIntBytes)(res.balance), &bitsetRaw)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}

	return res, bitsetFilter(bitsetRaw), nil
}

func (b *BalanceDB) updateBitset(asset *assetIdentity, block *big.Int, newBitset bitsetFilter) error {
	// Updating bitset value in place doesn't work.
	// Tried "INSERT INTO balance_history ... ON CONFLICT(chain_id, address, currency, block) DO UPDATE SET timestamp=excluded.timestamp, bitset=(bitset | excluded.bitset), balance=excluded.balance"
	_, err := b.db.Exec("UPDATE balance_history SET bitset = ? WHERE chain_id = ? AND address = ? AND currency = ? AND block = ?", int(newBitset), asset.ChainID, asset.Address, asset.TokenSymbol, (*bigint.SQLBigInt)(block))
	return err
}
