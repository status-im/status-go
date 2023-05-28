package transfer

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"

	"github.com/stretchr/testify/require"
)

type TestTransaction struct {
	Hash                 eth_common.Hash
	ChainID              common.ChainID
	From                 eth_common.Address // [sender]
	To                   eth_common.Address // [address]
	FromToken            string             // used to detect type in transfers table
	ToToken              string             // only used in multi_transactions table
	Timestamp            int64
	Value                int64
	BlkNumber            int64
	Success              bool
	MultiTransactionID   MultiTransactionIDType
	MultiTransactionType MultiTransactionType
}

func GenerateTestTransactions(t *testing.T, db *sql.DB, firstStartIndex int, count int) (result []TestTransaction, fromAddresses, toAddresses []eth_common.Address) {
	for i := firstStartIndex; i < (firstStartIndex + count); i++ {
		tr := TestTransaction{
			Hash:                 eth_common.HexToHash(fmt.Sprintf("0x1%d", i)),
			ChainID:              common.ChainID(i),
			From:                 eth_common.HexToAddress(fmt.Sprintf("0x2%d", i)),
			To:                   eth_common.HexToAddress(fmt.Sprintf("0x3%d", i)),
			Timestamp:            int64(i),
			Value:                int64(i),
			BlkNumber:            int64(i),
			Success:              true,
			MultiTransactionID:   NoMultiTransactionID,
			MultiTransactionType: MultiTransactionSend,
		}
		fromAddresses = append(fromAddresses, tr.From)
		toAddresses = append(toAddresses, tr.To)
		result = append(result, tr)
	}
	return
}

func InsertTestTransfer(t *testing.T, db *sql.DB, tr *TestTransaction) {
	// Respect `FOREIGN KEY(network_id,address,blk_hash)` of `transfers` table
	tokenType := "eth"
	if tr.FromToken != "" && strings.ToUpper(tr.FromToken) != testutils.EthSymbol {
		tokenType = "erc20"
	}
	blkHash := eth_common.HexToHash("4")
	_, err := db.Exec(`
		INSERT OR IGNORE INTO blocks(
			network_id, address, blk_number, blk_hash
		) VALUES (?, ?, ?, ?);
		INSERT INTO transfers (network_id, hash, address, blk_hash, tx,
			sender, receipt, log, type, blk_number, timestamp, loaded,
			multi_transaction_id, base_gas_fee, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, 0, ?)`,
		tr.ChainID, tr.To, tr.BlkNumber, blkHash,
		tr.ChainID, tr.Hash, tr.To, blkHash, &JSONBlob{}, tr.From, &JSONBlob{}, &JSONBlob{}, tokenType, tr.BlkNumber, tr.Timestamp, tr.MultiTransactionID, tr.Success)
	require.NoError(t, err)
}

func InsertTestPendingTransaction(t *testing.T, db *sql.DB, tr *TestTransaction) {
	_, err := db.Exec(`
		INSERT INTO pending_transactions (network_id, hash, timestamp, from_address, to_address,
			symbol, gas_price, gas_limit, value, data, type, additional_data, multi_transaction_id
		) VALUES (?, ?, ?, ?, ?, 'ETH', 0, 0, ?, '', 'test', '', ?)`,
		tr.ChainID, tr.Hash, tr.Timestamp, tr.From, tr.To, tr.Value, tr.MultiTransactionID)
	require.NoError(t, err)
}

func InsertTestMultiTransaction(t *testing.T, db *sql.DB, tr *TestTransaction) MultiTransactionIDType {
	fromTokenType := tr.FromToken
	if tr.FromToken == "" {
		fromTokenType = testutils.EthSymbol
	}
	toTokenType := tr.ToToken
	if tr.ToToken == "" {
		toTokenType = testutils.EthSymbol
	}
	result, err := db.Exec(`
		INSERT INTO multi_transactions (from_address, from_asset, from_amount, to_address, to_asset, type, timestamp
		) VALUES (?, ?, 0, ?, ?, ?, ?)`,
		tr.From, fromTokenType, tr.To, toTokenType, tr.MultiTransactionType, tr.Timestamp)
	require.NoError(t, err)
	rowID, err := result.LastInsertId()
	require.NoError(t, err)
	return MultiTransactionIDType(rowID)
}
