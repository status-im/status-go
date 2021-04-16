package wallet

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// DBHeader fields from header that are stored in database.
type DBHeader struct {
	Number         *big.Int
	Hash           common.Hash
	Timestamp      uint64
	Erc20Transfers []*Transfer
	Network        uint64
	Address        common.Address
	// Head is true if the block was a head at the time it was pulled from chain.
	Head bool
	// Loaded is true if trasfers from this block has been already fetched
	Loaded bool
}

func toDBHeader(header *types.Header) *DBHeader {
	return &DBHeader{
		Hash:      header.Hash(),
		Number:    header.Number,
		Timestamp: header.Time,
		Loaded:    false,
	}
}

// SyncOption is used to specify that application processed transfers for that block.
type SyncOption uint

// JSONBlob type for marshaling/unmarshaling inner type to json.
type JSONBlob struct {
	data interface{}
}

// Scan implements interface.
func (blob *JSONBlob) Scan(value interface{}) error {
	if value == nil || reflect.ValueOf(blob.data).IsNil() {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("not a byte slice")
	}
	if len(bytes) == 0 {
		return nil
	}
	err := json.Unmarshal(bytes, blob.data)
	return err
}

// Value implements interface.
func (blob *JSONBlob) Value() (driver.Value, error) {
	if blob.data == nil || reflect.ValueOf(blob.data).IsNil() {
		return nil, nil
	}
	return json.Marshal(blob.data)
}

func NewDB(db *sql.DB, network uint64) *Database {
	return &Database{db: db, network: network}
}

// Database sql wrapper for operations with wallet objects.
type Database struct {
	db      *sql.DB
	network uint64
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

func (db Database) ProcessBlocks(account common.Address, from *big.Int, to *LastKnownBlock, headers []*DBHeader) (err error) {
	var (
		tx *sql.Tx
	)

	tx, err = db.db.Begin()
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

	err = insertBlocksWithTransactions(tx, account, db.network, headers)
	if err != nil {
		return
	}

	err = upsertRange(tx, account, db.network, from, to)
	if err != nil {
		return
	}

	return
}

func (db Database) SaveBlocks(account common.Address, headers []*DBHeader) (err error) {
	var (
		tx *sql.Tx
	)
	tx, err = db.db.Begin()
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

	err = insertBlocksWithTransactions(tx, account, db.network, headers)
	if err != nil {
		return
	}

	return
}

// ProcessTranfers atomically adds/removes blocks and adds new tranfers.
func (db Database) ProcessTranfers(transfers []Transfer, removed []*DBHeader) (err error) {
	var (
		tx *sql.Tx
	)
	tx, err = db.db.Begin()
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
	err = deleteHeaders(tx, removed)
	if err != nil {
		return
	}
	err = updateOrInsertTransfers(tx, db.network, transfers)
	if err != nil {
		return
	}
	return
}

// SaveTranfers
func (db Database) SaveTranfers(address common.Address, transfers []Transfer, blocks []*big.Int) (err error) {
	var (
		tx *sql.Tx
	)
	tx, err = db.db.Begin()
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

	err = updateOrInsertTransfers(tx, db.network, transfers)
	if err != nil {
		return
	}

	err = markBlocksAsLoaded(tx, address, db.network, blocks)
	if err != nil {
		return
	}

	return
}

// GetTransfersInRange loads transfers for a given address between two blocks.
func (db *Database) GetTransfersInRange(address common.Address, start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(db.network).FilterAddress(address).FilterStart(start).FilterEnd(end).FilterLoaded(1)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfersByAddress loads transfers for a given address between two blocks.
func (db *Database) GetTransfersByAddress(address common.Address, toBlock *big.Int, limit int64) (rst []Transfer, err error) {
	query := newTransfersQuery().
		FilterNetwork(db.network).
		FilterAddress(address).
		FilterEnd(toBlock).
		FilterLoaded(1).
		Limit(limit)

	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetTransfersByAddressAndBlock loads transfers for a given address and block.
func (db *Database) GetTransfersByAddressAndBlock(address common.Address, block *big.Int, limit int64) (rst []Transfer, err error) {
	query := newTransfersQuery().
		FilterNetwork(db.network).
		FilterAddress(address).
		FilterBlockNumber(block).
		FilterLoaded(1).
		Limit(limit)

	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

// GetBlocksByAddress loads blocks for a given address.
func (db *Database) GetBlocksByAddress(address common.Address, limit int) (rst []*big.Int, err error) {
	query := `SELECT blk_number FROM blocks
	WHERE address = ? AND network_id = ? AND loaded = 0
	ORDER BY blk_number DESC 
	LIMIT ?`
	rows, err := db.db.Query(query, address, db.network, limit)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}
		rst = append(rst, block)
	}
	return rst, nil
}

func (db *Database) RemoveBlockWithTransfer(address common.Address, block *big.Int) error {
	query := `DELETE FROM blocks
	WHERE address = ? 
	AND blk_number = ? 
	AND network_id = ?`

	_, err := db.db.Exec(query, address, (*SQLBigInt)(block), db.network)

	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetLastBlockByAddress(address common.Address, limit int) (rst *big.Int, err error) {
	query := `SELECT * FROM 
	(SELECT blk_number FROM blocks WHERE address = ? AND network_id = ? ORDER BY blk_number DESC LIMIT ?)
	ORDER BY blk_number LIMIT 1`
	rows, err := db.db.Query(query, address, db.network, limit)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, nil
}

func (db *Database) GetLastSavedBlock() (rst *DBHeader, err error) {
	query := `SELECT blk_number, blk_hash 
	FROM blocks 
	WHERE network_id = ? 
	ORDER BY blk_number DESC LIMIT 1`
	rows, err := db.db.Query(query, db.network)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan((*SQLBigInt)(header.Number), &header.Hash)
		if err != nil {
			return nil, err
		}

		return header, nil
	}

	return nil, nil
}

func (db *Database) GetBlocks() (rst []*DBHeader, err error) {
	query := `SELECT blk_number, blk_hash, address FROM blocks`
	rows, err := db.db.Query(query, db.network)
	if err != nil {
		return
	}
	defer rows.Close()

	rst = []*DBHeader{}
	for rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan((*SQLBigInt)(header.Number), &header.Hash, &header.Address)
		if err != nil {
			return nil, err
		}

		rst = append(rst, header)
	}

	return rst, nil
}

func (db *Database) GetLastSavedBlockBefore(block *big.Int) (rst *DBHeader, err error) {
	query := `SELECT blk_number, blk_hash 
	FROM blocks 
	WHERE network_id = ? AND blk_number < ?
	ORDER BY blk_number DESC LIMIT 1`
	rows, err := db.db.Query(query, db.network, (*SQLBigInt)(block))
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		header := &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
		err = rows.Scan((*SQLBigInt)(header.Number), &header.Hash)
		if err != nil {
			return nil, err
		}

		return header, nil
	}

	return nil, nil
}

func (db *Database) GetFirstKnownBlock(address common.Address) (rst *big.Int, err error) {
	query := `SELECT blk_from FROM blocks_ranges
	WHERE address = ?
	AND network_id = ?
	ORDER BY blk_from
	LIMIT 1`

	rows, err := db.db.Query(query, address, db.network)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		block := &big.Int{}
		err = rows.Scan((*SQLBigInt)(block))
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	return nil, nil
}

type LastKnownBlock struct {
	Number  *big.Int
	Balance *big.Int
	Nonce   *int64
}

func (db *Database) GetLastKnownBlockByAddress(address common.Address) (block *LastKnownBlock, err error) {
	query := `SELECT blk_to, balance, nonce FROM blocks_ranges
	WHERE address = ?
	AND network_id = ?
	ORDER BY blk_to DESC
	LIMIT 1`

	rows, err := db.db.Query(query, address, db.network)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		var nonce sql.NullInt64
		block = &LastKnownBlock{Number: &big.Int{}, Balance: &big.Int{}}
		err = rows.Scan((*SQLBigInt)(block.Number), (*SQLBigIntBytes)(block.Balance), &nonce)
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

func (db *Database) GetLastKnownBlockByAddresses(addresses []common.Address) (map[common.Address]*LastKnownBlock, []common.Address, error) {
	res := map[common.Address]*LastKnownBlock{}
	accountsWithoutHistory := []common.Address{}
	for _, address := range addresses {
		block, err := db.GetLastKnownBlockByAddress(address)
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

// GetTransfers load transfers transfer betweeen two blocks.
func (db *Database) GetTransfers(start, end *big.Int) (rst []Transfer, err error) {
	query := newTransfersQuery().FilterNetwork(db.network).FilterStart(start).FilterEnd(end).FilterLoaded(1)
	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

func (db *Database) GetPreloadedTransactions(address common.Address, blockHash common.Hash) (rst []Transfer, err error) {
	query := newTransfersQuery().
		FilterNetwork(db.network).
		FilterAddress(address).
		FilterBlockHash(blockHash).
		FilterLoaded(0)

	rows, err := db.db.Query(query.String(), query.Args()...)
	if err != nil {
		return
	}
	defer rows.Close()
	return query.Scan(rows)
}

func (db *Database) GetTransactionsLog(address common.Address, transactionHash common.Hash) (*types.Log, error) {
	l := &types.Log{}
	err := db.db.QueryRow("SELECT log FROM transfers WHERE network_id = ? AND address = ? AND hash = ?",
		db.network, address, transactionHash).
		Scan(&JSONBlob{l})
	if err == nil {
		return l, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// SaveHeaders stores a list of headers atomically.
func (db *Database) SaveHeaders(headers []*types.Header, address common.Address) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	insert, err = tx.Prepare("INSERT INTO blocks(network_id, blk_number, blk_hash, address) VALUES (?, ?, ?, ?)")
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()

	for _, h := range headers {
		_, err = insert.Exec(db.network, (*SQLBigInt)(h.Number), h.Hash(), address)
		if err != nil {
			return
		}
	}
	return
}

// GetHeaderByNumber selects header using block number.
func (db *Database) GetHeaderByNumber(number *big.Int) (header *DBHeader, err error) {
	header = &DBHeader{Hash: common.Hash{}, Number: new(big.Int)}
	err = db.db.QueryRow("SELECT blk_hash, blk_number FROM blocks WHERE blk_number = ? AND network_id = ?", (*SQLBigInt)(number), db.network).Scan(&header.Hash, (*SQLBigInt)(header.Number))
	if err == nil {
		return header, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

type Token struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
	Symbol  string         `json:"symbol"`
	Color   string         `json:"color"`

	// Decimals defines how divisible the token is. For example, 0 would be
	// indivisible, whereas 18 would allow very small amounts of the token
	// to be traded.
	Decimals uint `json:"decimals"`
}

func (db *Database) GetCustomTokens() ([]*Token, error) {
	rows, err := db.db.Query(`SELECT address, name, symbol, decimals, color FROM tokens WHERE network_id = ?`, db.network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []*Token
	for rows.Next() {
		token := &Token{}
		err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.Color)
		if err != nil {
			return nil, err
		}

		rst = append(rst, token)
	}

	return rst, nil
}

func (db *Database) AddCustomToken(token Token) error {
	insert, err := db.db.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals, color) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(db.network, token.Address, token.Name, token.Symbol, token.Decimals, token.Color)
	return err
}

func (db *Database) DeleteCustomToken(address common.Address) error {
	_, err := db.db.Exec(`DELETE FROM TOKENS WHERE address = ?`, address)
	return err
}

type PendingTrxType string

const (
	RegisterENS    PendingTrxType = "RegisterENS"
	ReleaseENS     PendingTrxType = "ReleaseENS"
	SetPubKey      PendingTrxType = "SetPubKey"
	BuyStickerPack PendingTrxType = "BuyStickerPack"
	WalletTransfer PendingTrxType = "WalletTransfer"
)

type PendingTransaction struct {
	Hash           common.Hash    `json:"hash"`
	Timestamp      uint64         `json:"timestamp"`
	Value          BigInt         `json:"value"`
	From           common.Address `json:"from"`
	To             common.Address `json:"to"`
	Data           string         `json:"data"`
	Symbol         string         `json:"symbol"`
	GasPrice       BigInt         `json:"gasPrice"`
	GasLimit       BigInt         `json:"gasLimit"`
	Type           PendingTrxType `json:"type"`
	AdditionalData string         `json:"additionalData"`
}

func (db *Database) getAllPendingTransactions() ([]*PendingTransaction, error) {
	rows, err := db.db.Query(`SELECT hash, timestamp, value, from_address, to_address, data,
                                         symbol, gas_price, gas_limit, type, additional_data
                                  FROM pending_transactions
                                  WHERE network_id = ?`, db.network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*PendingTransaction
	for rows.Next() {
		transaction := &PendingTransaction{
			Value:    BigInt{Int: new(big.Int)},
			GasPrice: BigInt{Int: new(big.Int)},
			GasLimit: BigInt{Int: new(big.Int)},
		}
		err := rows.Scan(&transaction.Hash,
			&transaction.Timestamp,
			(*SQLBigIntBytes)(transaction.Value.Int),
			&transaction.From,
			&transaction.To,
			&transaction.Data,
			&transaction.Symbol,
			(*SQLBigIntBytes)(transaction.GasPrice.Int),
			(*SQLBigIntBytes)(transaction.GasLimit.Int),
			&transaction.Type,
			&transaction.AdditionalData,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (db *Database) getPendingOutboundTransactionsByAddress(address common.Address) ([]*PendingTransaction, error) {
	rows, err := db.db.Query(`SELECT hash, timestamp, value, from_address, to_address, data,
                                         symbol, gas_price, gas_limit, type, additional_data
                                  FROM pending_transactions
                                  WHERE network_id = ?
                                  AND from_address = ?`, db.network, address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*PendingTransaction
	for rows.Next() {
		transaction := &PendingTransaction{
			Value:    BigInt{Int: new(big.Int)},
			GasPrice: BigInt{Int: new(big.Int)},
			GasLimit: BigInt{Int: new(big.Int)},
		}
		err := rows.Scan(&transaction.Hash,
			&transaction.Timestamp,
			(*SQLBigIntBytes)(transaction.Value.Int),
			&transaction.From,
			&transaction.To,
			&transaction.Data,
			&transaction.Symbol,
			(*SQLBigIntBytes)(transaction.GasPrice.Int),
			(*SQLBigIntBytes)(transaction.GasLimit.Int),
			&transaction.Type,
			&transaction.AdditionalData,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (db *Database) addPendingTransaction(transaction PendingTransaction) error {
	insert, err := db.db.Prepare(`INSERT OR REPLACE INTO pending_transactions
                                      (network_id, hash, timestamp, value, from_address, to_address,
                                       data, symbol, gas_price, gas_limit, type, additional_data)
                                      VALUES
                                      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	_, err = insert.Exec(db.network,
		transaction.Hash,
		transaction.Timestamp,
		(*SQLBigIntBytes)(transaction.Value.Int),
		transaction.From,
		transaction.To,
		transaction.Data,
		transaction.Symbol,
		(*SQLBigIntBytes)(transaction.GasPrice.Int),
		(*SQLBigIntBytes)(transaction.GasLimit.Int),
		transaction.Type,
		transaction.AdditionalData,
	)
	return err
}

func (db *Database) deletePendingTransaction(hash common.Hash) error {
	_, err := db.db.Exec(`DELETE FROM pending_transactions WHERE hash = ?`, hash)
	return err
}

type Favourite struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
}

func (db *Database) GetFavourites() ([]*Favourite, error) {
	rows, err := db.db.Query(`SELECT address, name FROM favourites`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []*Favourite
	for rows.Next() {
		favourite := &Favourite{}
		err := rows.Scan(&favourite.Address, &favourite.Name)
		if err != nil {
			return nil, err
		}

		rst = append(rst, favourite)
	}

	return rst, nil
}

func (db *Database) AddFavourite(favourite Favourite) error {
	insert, err := db.db.Prepare("INSERT OR REPLACE INTO favourites (address, name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(favourite.Address, favourite.Name)
	return err
}

// statementCreator allows to pass transaction or database to use in consumer.
type statementCreator interface {
	Prepare(query string) (*sql.Stmt, error)
}

func deleteHeaders(creator statementCreator, headers []*DBHeader) error {
	delete, err := creator.Prepare("DELETE FROM blocks WHERE blk_hash = ?")
	if err != nil {
		return err
	}
	deleteTransfers, err := creator.Prepare("DELETE FROM transfers WHERE blk_hash = ?")
	if err != nil {
		return err
	}
	for _, h := range headers {
		_, err = delete.Exec(h.Hash)
		if err != nil {
			return err
		}

		_, err = deleteTransfers.Exec(h.Hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertBlocksWithTransactions(creator statementCreator, account common.Address, network uint64, headers []*DBHeader) error {
	insert, err := creator.Prepare("INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash, loaded) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	updateTx, err := creator.Prepare(`UPDATE transfers 
	SET log = ? 
	WHERE network_id = ? AND address = ? AND hash = ?`)
	if err != nil {
		return err
	}

	insertTx, err := creator.Prepare(`INSERT OR IGNORE 
	INTO transfers (network_id, address, sender, hash, blk_number, blk_hash, type, timestamp, log, loaded)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return err
	}

	for _, header := range headers {
		_, err = insert.Exec(network, account, (*SQLBigInt)(header.Number), header.Hash, header.Loaded)
		if err != nil {
			return err
		}
		if len(header.Erc20Transfers) > 0 {
			for _, transfer := range header.Erc20Transfers {
				res, err := updateTx.Exec(&JSONBlob{transfer.Log}, network, account, transfer.ID)
				if err != nil {
					return err
				}
				affected, err := res.RowsAffected()
				if err != nil {
					return err
				}
				if affected > 0 {
					continue
				}

				_, err = insertTx.Exec(network, account, account, transfer.ID, (*SQLBigInt)(header.Number), header.Hash, erc20Transfer, transfer.Timestamp, &JSONBlob{transfer.Log})
				if err != nil {
					log.Error("error saving erc20transfer", "err", err)
					return err
				}
			}
		}
	}
	return nil
}

func (db *Database) UpsertRange(account common.Address, network uint64, from, to, balance *big.Int, nonce uint64) error {
	log.Debug("upsert blocks range", "account", account, "network id", network, "from", from, "to", to, "balance", balance, "nonce", nonce)
	insert, err := db.db.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to, balance, nonce) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(network, account, (*SQLBigInt)(from), (*SQLBigInt)(to), (*SQLBigIntBytes)(balance), &nonce)
	return err
}

func upsertRange(creator statementCreator, account common.Address, network uint64, from *big.Int, to *LastKnownBlock) (err error) {
	log.Debug("upsert blocks range", "account", account, "network id", network, "from", from, "to", to.Number, "balance", to.Balance)
	update, err := creator.Prepare(`UPDATE blocks_ranges
                SET blk_to = ?, balance = ?, nonce = ?
                WHERE address = ?
                AND network_id = ?
                AND blk_to = ?`)

	if err != nil {
		return err
	}

	res, err := update.Exec((*SQLBigInt)(to.Number), (*SQLBigIntBytes)(to.Balance), to.Nonce, account, network, (*SQLBigInt)(from))

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

		_, err = insert.Exec(network, account, (*SQLBigInt)(from), (*SQLBigInt)(to.Number), (*SQLBigIntBytes)(to.Balance), to.Nonce)
		if err != nil {
			return err
		}
	}

	return
}

type BlocksRange struct {
	from *big.Int
	to   *big.Int
}

func (db *Database) getOldRanges(account common.Address, network uint64) ([]*BlocksRange, error) {
	query := `select blk_from, blk_to from blocks_ranges
	          where address = ?
	          and network_id = ?
	          order by blk_from`

	rows, err := db.db.Query(query, account, db.network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ranges := []*BlocksRange{}
	for rows.Next() {
		from := &big.Int{}
		to := &big.Int{}
		err = rows.Scan((*SQLBigInt)(from), (*SQLBigInt)(to))
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

func (db *Database) mergeRanges(account common.Address, network uint64) (err error) {
	var (
		tx *sql.Tx
	)

	ranges, err := db.getOldRanges(account, network)
	if err != nil {
		return err
	}

	log.Info("merge old ranges", "account", account, "network", network, "ranges", len(ranges))

	if len(ranges) <= 1 {
		return nil
	}

	tx, err = db.db.Begin()
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
		err = deleteRange(tx, account, network, rangeToDelete.from, rangeToDelete.to)
		if err != nil {
			return err
		}
	}

	for _, newRange := range newRanges {
		err = insertRange(tx, account, network, newRange.from, newRange.to)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteRange(creator statementCreator, account common.Address, network uint64, from *big.Int, to *big.Int) error {
	log.Info("delete blocks range", "account", account, "network", network, "from", from, "to", to)
	delete, err := creator.Prepare(`DELETE FROM blocks_ranges
                                        WHERE address = ?
                                        AND network_id = ?
                                        AND blk_from = ?
                                        AND blk_to = ?`)
	if err != nil {
		log.Info("some error", "error", err)
		return err
	}

	_, err = delete.Exec(account, network, (*SQLBigInt)(from), (*SQLBigInt)(to))
	return err
}

func insertRange(creator statementCreator, account common.Address, network uint64, from *big.Int, to *big.Int) error {
	log.Info("insert blocks range", "account", account, "network", network, "from", from, "to", to)
	insert, err := creator.Prepare("INSERT INTO blocks_ranges (network_id, address, blk_from, blk_to) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = insert.Exec(network, account, (*SQLBigInt)(from), (*SQLBigInt)(to))
	return err
}

func updateOrInsertTransfers(creator statementCreator, network uint64, transfers []Transfer) error {
	update, err := creator.Prepare(`UPDATE transfers
	SET tx = ?, sender = ?, receipt = ?, timestamp = ?, loaded = 1
	WHERE address =?  AND hash = ?`)
	if err != nil {
		return err
	}

	insert, err := creator.Prepare(`INSERT OR IGNORE INTO transfers
	(network_id, hash, blk_hash, blk_number, timestamp, address, tx, sender, receipt, log, type, loaded)
	VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return err
	}
	for _, t := range transfers {
		res, err := update.Exec(&JSONBlob{t.Transaction}, t.From, &JSONBlob{t.Receipt}, t.Timestamp, t.Address, t.ID)

		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected > 0 {
			continue
		}

		_, err = insert.Exec(network, t.ID, t.BlockHash, (*SQLBigInt)(t.BlockNumber), t.Timestamp, t.Address, &JSONBlob{t.Transaction}, t.From, &JSONBlob{t.Receipt}, &JSONBlob{t.Log}, t.Type)
		if err != nil {
			log.Error("can't save transfer", "b-hash", t.BlockHash, "b-n", t.BlockNumber, "a", t.Address, "h", t.ID)
			return err
		}
	}
	return nil
}

//markBlocksAsLoaded(tx, address, db.network, blocks)
func markBlocksAsLoaded(creator statementCreator, address common.Address, network uint64, blocks []*big.Int) error {
	update, err := creator.Prepare("UPDATE blocks SET loaded=? WHERE address=? AND blk_number=? AND network_id=?")
	if err != nil {
		return err
	}

	for _, block := range blocks {
		_, err := update.Exec(true, address, (*SQLBigInt)(block), network)
		if err != nil {
			return err
		}
	}
	return nil
}

type BigInt struct {
	*big.Int
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	return []byte("\"" + b.String() + "\""), nil
}

func (b *BigInt) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	z := new(big.Int)
	_, ok := z.SetString(strings.Trim(string(p), "\""), 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", "123")
	}
	b.Int = z
	return nil
}
