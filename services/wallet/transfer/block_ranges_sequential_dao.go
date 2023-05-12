package transfer

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/bigint"
)

const (
	firstBlockColumn = "blk_first"
	lastBlockColumn  = "blk_last"
	startBlockColumn = "blk_start"
)

type BlockRangeSequentialDAO struct {
	db *sql.DB
}

type BlockRange struct {
	Start      *big.Int // Block of first transfer
	FirstKnown *big.Int // Oldest scanned block
	LastKnown  *big.Int // Last scanned block
}

func NewBlockRange() *BlockRange {
	return &BlockRange{Start: &big.Int{}, FirstKnown: &big.Int{}, LastKnown: &big.Int{}}
}

func (b *BlockRangeSequentialDAO) getBlockRange(chainID uint64, address common.Address) (blockRange *BlockRange, err error) {
	query := `SELECT blk_start, blk_first, blk_last FROM blocks_ranges_sequential
	WHERE address = ?
	AND network_id = ?`

	rows, err := b.db.Query(query, address, chainID)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		blockRange = NewBlockRange()
		err = rows.Scan((*bigint.SQLBigInt)(blockRange.Start), (*bigint.SQLBigInt)(blockRange.FirstKnown), (*bigint.SQLBigInt)(blockRange.LastKnown))
		if err != nil {
			return nil, err
		}

		return blockRange, nil
	}

	return nil, nil
}

// TODO call it when account is removed
//
//lint:ignore U1000 Ignore unused function temporarily
func (b *BlockRangeSequentialDAO) deleteRange(chainID uint64, account common.Address) error {
	log.Info("delete blocks range", "account", account, "network", chainID)
	delete, err := b.db.Prepare(`DELETE FROM blocks_ranges_sequential
                                        WHERE address = ?
                                        AND network_id = ?`)
	if err != nil {
		log.Info("some error", "error", err)
		return err
	}

	_, err = delete.Exec(account, chainID)
	return err
}

func (b *BlockRangeSequentialDAO) updateStartBlock(chainID uint64, account common.Address, block *big.Int) (err error) {
	return updateBlock(b.db, chainID, account, startBlockColumn, block)
}

//lint:ignore U1000 Ignore unused function temporarily, TODO use it when new transfers are fetched
func (b *BlockRangeSequentialDAO) updateLastBlock(chainID uint64, account common.Address, block *big.Int) (err error) {
	return updateBlock(b.db, chainID, account, lastBlockColumn, block)
}

func (b *BlockRangeSequentialDAO) updateFirstBlock(chainID uint64, account common.Address, block *big.Int) (err error) {
	return updateBlock(b.db, chainID, account, firstBlockColumn, block)
}

func updateBlock(creator statementCreator, chainID uint64, account common.Address,
	blockColumn string, block *big.Int) (err error) {

	update, err := creator.Prepare(fmt.Sprintf(`UPDATE blocks_ranges_sequential
                SET %s = ?
                WHERE address = ?
                AND network_id = ?`, blockColumn))

	if err != nil {
		return err
	}

	_, err = update.Exec((*bigint.SQLBigInt)(block), account, chainID)

	if err != nil {
		return err
	}

	return
}

func (b *BlockRangeSequentialDAO) upsertRange(chainID uint64, account common.Address,
	start *big.Int, first *big.Int, last *big.Int) (err error) {

	log.Info("upsert blocks range", "account", account, "network id", chainID, "start", start, "first", first, "last", last)

	update, err := b.db.Prepare(`UPDATE blocks_ranges_sequential
                SET blk_start = ?,
                blk_first = ?,
                blk_last = ?
                WHERE address = ?
                AND network_id = ?`)

	if err != nil {
		return err
	}

	res, err := update.Exec((*bigint.SQLBigInt)(start), (*bigint.SQLBigInt)(first), (*bigint.SQLBigInt)(last), account, chainID)

	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		insert, err := b.db.Prepare("INSERT INTO blocks_ranges_sequential (network_id, address, blk_first, blk_last, blk_start) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			return err
		}

		_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(first), (*bigint.SQLBigInt)(last), (*bigint.SQLBigInt)(start))
		if err != nil {
			return err
		}
	}

	return
}
