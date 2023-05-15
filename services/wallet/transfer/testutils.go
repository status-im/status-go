package transfer

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

type TestTransaction struct {
	Hash                 common.Hash
	ChainID              uint64
	From                 common.Address // [sender]
	To                   common.Address // [address]
	Timestamp            int64
	Value                int64
	BlkNumber            int64
	MultiTransactionID   MultiTransactionIDType
	MultiTransactionType MultiTransactionType
}

func GenerateTestTransactions(t *testing.T, db *sql.DB, firstStartIndex int, count int) (result []TestTransaction) {
	for i := firstStartIndex; i < (firstStartIndex + count); i++ {
		tr := TestTransaction{
			Hash:                 common.HexToHash(fmt.Sprintf("0x1%d", i)),
			ChainID:              uint64(i),
			From:                 common.HexToAddress(fmt.Sprintf("0x2%d", i)),
			To:                   common.HexToAddress(fmt.Sprintf("0x3%d", i)),
			Timestamp:            int64(i),
			Value:                int64(i),
			BlkNumber:            int64(i),
			MultiTransactionID:   NoMultiTransactionID,
			MultiTransactionType: MultiTransactionSend,
		}
		result = append(result, tr)
	}
	return
}

func InsertTestTransfer(t *testing.T, db *sql.DB, tr *TestTransaction) {
	// Respect `FOREIGN KEY(network_id,address,blk_hash)` of `transfers` table
	blkHash := common.HexToHash("4")
	_, err := db.Exec(`
		INSERT OR IGNORE INTO blocks(
			network_id, address, blk_number, blk_hash
		) VALUES (?, ?, ?, ?);
		INSERT INTO transfers (network_id, hash, address, blk_hash, tx,
			sender, receipt, log, type, blk_number, timestamp, loaded,
			multi_transaction_id, base_gas_fee
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, "test", ?, ?, 0, ?, 0)`,
		tr.ChainID, tr.To, tr.BlkNumber, blkHash,
		tr.ChainID, tr.Hash, tr.To, blkHash, &JSONBlob{}, tr.From, &JSONBlob{}, &JSONBlob{}, tr.BlkNumber, tr.Timestamp, tr.MultiTransactionID)
	require.NoError(t, err)
}

func InsertTestMultiTransaction(t *testing.T, db *sql.DB, tr *TestTransaction) MultiTransactionIDType {
	result, err := db.Exec(`
		INSERT INTO multi_transactions (from_address, from_asset, from_amount, to_address, to_asset, type, timestamp
		) VALUES (?, 'ETH', 0, ?, 'SNT', ?, ?)`,
		tr.From, tr.To, tr.MultiTransactionType, tr.Timestamp)
	require.NoError(t, err)
	rowID, err := result.LastInsertId()
	require.NoError(t, err)
	return MultiTransactionIDType(rowID)
}
