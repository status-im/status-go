package transfer

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/chain"
)

type BlocksRange struct {
	from *big.Int
	to   *big.Int
}

type LastKnownBlock struct {
	Number  *big.Int
	Balance *big.Int
	Nonce   *int64
}

type LastKnownBlockView struct {
	Address common.Address `json:"address"`
	Number  *big.Int       `json:"blockNumber"`
	Balance bigint.BigInt  `json:"balance"`
	Nonce   *int64         `json:"nonce"`
}

func blocksToViews(blocks map[common.Address]*LastKnownBlock) []LastKnownBlockView {
	blocksViews := []LastKnownBlockView{}
	for address, block := range blocks {
		view := LastKnownBlockView{
			Address: address,
			Number:  block.Number,
			Balance: bigint.BigInt{block.Balance},
			Nonce:   block.Nonce,
		}
		blocksViews = append(blocksViews, view)
	}

	return blocksViews
}

type Block struct {
	db *sql.DB
}

// MergeBlocksRanges merge old blocks ranges if possible
func (b *Block) mergeBlocksRanges(chainIDs []uint64, accounts []common.Address) error {
	for _, chainID := range chainIDs {
		for _, account := range accounts {
			err := b.mergeRanges(chainID, account)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Block) setInitialBlocksRange(chainClient *chain.Client) error {
	accountsDB, err := accounts.NewDB(b.db)
	if err != nil {
		return err
	}
	watchAddress, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}

	from := big.NewInt(0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	header, err := chainClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	err = b.insertRange(chainClient.ChainID, common.Address(watchAddress), from, header.Number, big.NewInt(0), 0)
	if err != nil {
		return err
	}
	return nil
}

func (b *Block) mergeRanges(chainID uint64, account common.Address) (err error) {
	var (
		tx *sql.Tx
	)

	ranges, err := b.getOldRanges(chainID, account)
	if err != nil {
		return err
	}

	log.Info("merge old ranges", "account", account, "network", chainID, "ranges", len(ranges))

	if len(ranges) <= 1 {
		return nil
	}

	tx, err = b.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	newRanges, deletedRanges := getNewRanges(ranges)

	for _, rangeToDelete := range deletedRanges {
		err = deleteRange(chainID, tx, account, rangeToDelete.from, rangeToDelete.to)
		if err != nil {
			return err
		}
	}

	for _, newRange := range newRanges {
		err = insertRange(chainID, tx, account, newRange.from, newRange.to)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Block) insertRange(chainID uint64, account common.Address, from, to, balance *big.Int, nonce uint64) error {
	log.Debug("insert blocks range", "account", account, "network id", chainID, "from", from, "to", to, "balance", balance, "nonce", nonce)
	insert, err := b.db.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to, balance, nonce) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(from), (*bigint.SQLBigInt)(to), (*bigint.SQLBigIntBytes)(balance), &nonce)
	return err
}

func (b *Block) getOldRanges(chainID uint64, account common.Address) ([]*BlocksRange, error) {
	query := `select blk_from, blk_to from blocks_ranges
	          where address = ?
	          and network_id = ?
	          order by blk_from`

	rows, err := b.db.Query(query, account, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ranges := []*BlocksRange{}
	for rows.Next() {
		from := &big.Int{}
		to := &big.Int{}
		err = rows.Scan((*bigint.SQLBigInt)(from), (*bigint.SQLBigInt)(to))
		if err != nil {
			return nil, err
		}

		ranges = append(ranges, &BlocksRange{
			from: from,
			to:   to,
		})
	}

	return ranges, nil
}

// GetBlocksByAddress loads blocks for a given address.
func (b *Block) GetBlocksByAddress(chainID uint64, address common.Address, limit int) (rst []*big.Int, err error) {
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

func (b *Block) RemoveBlockWithTransfer(chainID uint64, address common.Address, block *big.Int) error {
	query := `DELETE FROM blocks
	WHERE address = ? 
	AND blk_number = ? 
	AND network_id = ?`

	_, err := b.db.Exec(query, address, (*bigint.SQLBigInt)(block), chainID)

	if err != nil {
		return err
	}

	return nil
}

func (b *Block) GetLastBlockByAddress(chainID uint64, address common.Address, limit int) (rst *big.Int, err error) {
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

func (b *Block) GetLastSavedBlock(chainID uint64) (rst *DBHeader, err error) {
	query := `SELECT blk_number, blk_hash 
	FROM blocks 
	WHERE network_id = ? 
	ORDER BY blk_number DESC LIMIT 1`
	rows, err := b.db.Query(query, chainID)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan((*bigint.SQLBigInt)(header.Number), &header.Hash)
		if err != nil {
			return nil, err
		}

		return header, nil
	}

	return nil, nil
}

func (b *Block) GetBlocks(chainID uint64) (rst []*DBHeader, err error) {
	query := `SELECT blk_number, blk_hash, address FROM blocks`
	rows, err := b.db.Query(query, chainID)
	if err != nil {
		return
	}
	defer rows.Close()

	rst = []*DBHeader{}
	for rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan((*bigint.SQLBigInt)(header.Number), &header.Hash, &header.Address)
		if err != nil {
			return nil, err
		}

		rst = append(rst, header)
	}

	return rst, nil
}

func (b *Block) GetLastSavedBlockBefore(chainID uint64, block *big.Int) (rst *DBHeader, err error) {
	query := `SELECT blk_number, blk_hash 
	FROM blocks 
	WHERE network_id = ? AND blk_number < ?
	ORDER BY blk_number DESC LIMIT 1`
	rows, err := b.db.Query(query, chainID, (*bigint.SQLBigInt)(block))
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan((*bigint.SQLBigInt)(header.Number), &header.Hash)
		if err != nil {
			return nil, err
		}

		return header, nil
	}

	return nil, nil
}

func (b *Block) GetFirstKnownBlock(chainID uint64, address common.Address) (rst *big.Int, err error) {
	query := `SELECT blk_from FROM blocks_ranges
	WHERE address = ?
	AND network_id = ?
	ORDER BY blk_from
	LIMIT 1`

	rows, err := b.db.Query(query, address, chainID)
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

func (b *Block) GetLastKnownBlockByAddress(chainID uint64, address common.Address) (block *LastKnownBlock, err error) {
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
		block = &LastKnownBlock{Number: &big.Int{}, Balance: &big.Int{}}
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

func (b *Block) getLastKnownBalances(chainID uint64, addresses []common.Address) (map[common.Address]*LastKnownBlock, error) {
	result := map[common.Address]*LastKnownBlock{}
	for _, address := range addresses {
		block, error := b.GetLastKnownBlockByAddress(chainID, address)
		if error != nil {
			return nil, error
		}

		if block != nil {
			result[address] = block
		}
	}

	return result, nil
}

func (b *Block) GetLastKnownBlockByAddresses(chainID uint64, addresses []common.Address) (map[common.Address]*LastKnownBlock, []common.Address, error) {
	res := map[common.Address]*LastKnownBlock{}
	accountsWithoutHistory := []common.Address{}
	for _, address := range addresses {
		block, err := b.GetLastKnownBlockByAddress(chainID, address)
		if err != nil {
			log.Info("Can't get last block", "error", err)
			return nil, nil, err
		}

		if block != nil {
			res[address] = block
		} else {
			accountsWithoutHistory = append(accountsWithoutHistory, address)
		}
	}

	return res, accountsWithoutHistory, nil
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
			log.Info("blocks ranges gap detected", "from", prevTo, "to", blocksRange.from)
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

func deleteRange(chainID uint64, creator statementCreator, account common.Address, from *big.Int, to *big.Int) error {
	log.Info("delete blocks range", "account", account, "network", chainID, "from", from, "to", to)
	delete, err := creator.Prepare(`DELETE FROM blocks_ranges
                                        WHERE address = ?
                                        AND network_id = ?
                                        AND blk_from = ?
                                        AND blk_to = ?`)
	if err != nil {
		log.Info("some error", "error", err)
		return err
	}

	_, err = delete.Exec(account, chainID, (*bigint.SQLBigInt)(from), (*bigint.SQLBigInt)(to))
	return err
}

func insertRange(chainID uint64, creator statementCreator, account common.Address, from *big.Int, to *big.Int) error {
	log.Info("insert blocks range", "account", account, "network", chainID, "from", from, "to", to)
	insert, err := creator.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(from), (*bigint.SQLBigInt)(to))
	return err
}

func upsertRange(chainID uint64, creator statementCreator, account common.Address, from *big.Int, to *LastKnownBlock) (err error) {
	log.Debug("upsert blocks range", "account", account, "network id", chainID, "from", from, "to", to.Number, "balance", to.Balance)
	update, err := creator.Prepare(`UPDATE blocks_ranges
                SET blk_to = ?, balance = ?, nonce = ?
                WHERE address = ?
                AND network_id = ?
                AND blk_to = ?`)

	if err != nil {
		return err
	}

	res, err := update.Exec((*bigint.SQLBigInt)(to.Number), (*bigint.SQLBigIntBytes)(to.Balance), to.Nonce, account, chainID, (*bigint.SQLBigInt)(from))

	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		insert, err := creator.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to, balance, nonce) VALUES (?, ?, ?, ?, ?, ?)")
		if err != nil {
			return err
		}

		_, err = insert.Exec(chainID, account, (*bigint.SQLBigInt)(from), (*bigint.SQLBigInt)(to.Number), (*bigint.SQLBigIntBytes)(to.Balance), to.Nonce)
		if err != nil {
			return err
		}
	}

	return
}
