package transfer

import (
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/bigint"
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

func (b *BlockRangeSequentialDAO) deleteRange(account common.Address) error {
	log.Debug("delete blocks range", "account", account)
	delete, err := b.db.Prepare(`DELETE FROM blocks_ranges_sequential WHERE address = ?`)
	if err != nil {
		log.Error("Failed to prepare deletion of sequential block range", "error", err)
		return err
	}

	_, err = delete.Exec(account)
	return err
}

func (b *BlockRangeSequentialDAO) upsertRange(chainID uint64, account common.Address,
	newBlockRange *BlockRange) (err error) {

	log.Debug("upsert blocks range", "account", account, "chainID", chainID,
		"start", newBlockRange.Start, "first", newBlockRange.FirstKnown, "last", newBlockRange.LastKnown)

	blockRange, err := b.getBlockRange(chainID, account)
	if err != nil {
		return err
	}

	// Update existing range
	if blockRange != nil {
		// Ovewrite start block if there was not any or if new one is older, because it can be precised only
		// to a greater value, because no history can be before some block that is considered
		// as a start of history, but due to concurrent block range checks, a newer greater block
		// can be found that matches criteria of a start block (nonce is zero, balances are equal)
		if newBlockRange.Start != nil && (blockRange.Start == nil || blockRange.Start.Cmp(newBlockRange.Start) < 0) {
			blockRange.Start = newBlockRange.Start
		}

		// Overwrite first known block if there was not any or if new one is older
		if (blockRange.FirstKnown == nil && newBlockRange.FirstKnown != nil) ||
			(blockRange.FirstKnown != nil && newBlockRange.FirstKnown != nil && blockRange.FirstKnown.Cmp(newBlockRange.FirstKnown) > 0) {
			blockRange.FirstKnown = newBlockRange.FirstKnown
		}

		// Overwrite last known block if there was not any or if new one is newer
		if (blockRange.LastKnown == nil && newBlockRange.LastKnown != nil) ||
			(blockRange.LastKnown != nil && newBlockRange.LastKnown != nil && blockRange.LastKnown.Cmp(newBlockRange.LastKnown) < 0) {
			blockRange.LastKnown = newBlockRange.LastKnown
		}

		log.Debug("update blocks range", "account", account, "chainID", chainID,
			"start", blockRange.Start, "first", blockRange.FirstKnown, "last", blockRange.LastKnown)
	} else {
		blockRange = newBlockRange
	}

	upsert, err := b.db.Prepare(`REPLACE INTO blocks_ranges_sequential
					(network_id, address, blk_start, blk_first, blk_last) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}

	_, err = upsert.Exec(chainID, account, (*bigint.SQLBigInt)(blockRange.Start), (*bigint.SQLBigInt)(blockRange.FirstKnown),
		(*bigint.SQLBigInt)(blockRange.LastKnown))

	return
}
