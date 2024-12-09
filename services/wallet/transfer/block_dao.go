package transfer

import (
	"database/sql"
	"math/big"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type BlocksRange struct {
	from *big.Int
	to   *big.Int
}

type Block struct {
	Number  *big.Int
	Balance *big.Int
	Nonce   *int64
}

type BlockDAO struct {
	db *sql.DB
}

func (b *BlockDAO) insertRange(chainID uint64, account common.Address, from, to, balance *big.Int, nonce uint64) error {
	logutils.ZapLogger().Debug(
		"insert blocks range",
		zap.Stringer("account", account),
		zap.Uint64("network id", chainID),
		zap.Stringer("from", from),
		zap.Stringer("to", to),
		zap.Stringer("balance", balance),
		zap.Uint64("nonce", nonce),
	)
	insert, err := b.db.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to, balance, nonce) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(from), (*bigint.SQLBigInt)(to), (*bigint.SQLBigIntBytes)(balance), &nonce)
	return err
}

// GetBlocksToLoadByAddress gets unloaded blocks for a given address.
func (b *BlockDAO) GetBlocksToLoadByAddress(chainID uint64, address common.Address, limit int) (rst []*big.Int, err error) {
	query := `SELECT blk_number FROM blocks
	WHERE address = ? AND network_id = ? AND loaded = 0
	ORDER BY blk_number DESC
	LIMIT ?`
	rows, err := b.db.Query(query, address, chainID, limit)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*bigint.SQLBigInt)(block))
		if err != nil {
			return nil, err
		}
		rst = append(rst, block)
	}
	return rst, nil
}

func (b *BlockDAO) GetLastBlockByAddress(chainID uint64, address common.Address, limit int) (rst *big.Int, err error) {
	query := `SELECT * FROM
	(SELECT blk_number FROM blocks WHERE address = ? AND network_id = ? ORDER BY blk_number DESC LIMIT ?)
	ORDER BY blk_number LIMIT 1`
	rows, err := b.db.Query(query, address, chainID, limit)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*bigint.SQLBigInt)(block))
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, nil
}

func (b *BlockDAO) GetLastKnownBlockByAddress(chainID uint64, address common.Address) (block *Block, err error) {
	query := `SELECT blk_to, balance, nonce FROM blocks_ranges
	WHERE address = ?
	AND network_id = ?
	ORDER BY blk_to DESC
	LIMIT 1`

	rows, err := b.db.Query(query, address, chainID)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		var nonce sql.NullInt64
		block = &Block{Number: &big.Int{}, Balance: &big.Int{}}
		err = rows.Scan((*bigint.SQLBigInt)(block.Number), (*bigint.SQLBigIntBytes)(block.Balance), &nonce)
		if err != nil {
			return nil, err
		}

		if nonce.Valid {
			block.Nonce = &nonce.Int64
		}
		return block, nil
	}

	return nil, nil
}

func getNewRanges(ranges []*BlocksRange) ([]*BlocksRange, []*BlocksRange) {
	initValue := big.NewInt(-1)
	prevFrom := big.NewInt(-1)
	prevTo := big.NewInt(-1)
	hasMergedRanges := false
	var newRanges []*BlocksRange
	var deletedRanges []*BlocksRange
	for idx, blocksRange := range ranges {
		if prevTo.Cmp(initValue) == 0 {
			prevTo = blocksRange.to
			prevFrom = blocksRange.from
		} else if prevTo.Cmp(blocksRange.from) >= 0 {
			hasMergedRanges = true
			deletedRanges = append(deletedRanges, ranges[idx-1])
			if prevTo.Cmp(blocksRange.to) <= 0 {
				prevTo = blocksRange.to
			}
		} else {
			if hasMergedRanges {
				deletedRanges = append(deletedRanges, ranges[idx-1])
				newRanges = append(newRanges, &BlocksRange{
					from: prevFrom,
					to:   prevTo,
				})
			}
			logutils.ZapLogger().Info("blocks ranges gap detected",
				zap.Stringer("from", prevTo),
				zap.Stringer("to", blocksRange.from),
			)
			hasMergedRanges = false

			prevFrom = blocksRange.from
			prevTo = blocksRange.to
		}
	}

	if hasMergedRanges {
		deletedRanges = append(deletedRanges, ranges[len(ranges)-1])
		newRanges = append(newRanges, &BlocksRange{
			from: prevFrom,
			to:   prevTo,
		})
	}

	return newRanges, deletedRanges
}

func deleteAllRanges(creator statementCreator, account common.Address) error {
	delete, err := creator.Prepare(`DELETE FROM blocks_ranges WHERE address = ?`)
	if err != nil {
		return err
	}

	_, err = delete.Exec(account)
	return err
}
