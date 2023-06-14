package transfer

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/sqlite"

	"github.com/stretchr/testify/require"
)

type TestTransaction struct {
	Hash               eth_common.Hash
	ChainID            common.ChainID
	From               eth_common.Address // [sender]
	Timestamp          int64
	BlkNumber          int64
	Success            bool
	MultiTransactionID MultiTransactionIDType
}

type TestTransfer struct {
	TestTransaction
	To    eth_common.Address // [address]
	Token string             // used to detect type in transfers table
	Value int64
}

type TestMultiTransaction struct {
	MultiTransactionID   MultiTransactionIDType
	MultiTransactionType MultiTransactionType
	FromAddress          eth_common.Address
	ToAddress            eth_common.Address
	FromToken            string
	ToToken              string
	FromAmount           int64
	ToAmount             int64
	Timestamp            int64
}

func generateTestTransaction(seed int) TestTransaction {
	return TestTransaction{
		Hash:               eth_common.HexToHash(fmt.Sprintf("0x1%d", seed)),
		ChainID:            common.ChainID(seed),
		From:               eth_common.HexToAddress(fmt.Sprintf("0x2%d", seed)),
		Timestamp:          int64(seed),
		BlkNumber:          int64(seed),
		Success:            true,
		MultiTransactionID: NoMultiTransactionID,
	}
}

func generateTestTransfer(seed int) TestTransfer {
	return TestTransfer{
		TestTransaction: generateTestTransaction(seed),
		Token:           "",
		To:              eth_common.HexToAddress(fmt.Sprintf("0x3%d", seed)),
		Value:           int64(seed),
	}
}

func GenerateTestSendMultiTransaction(tr TestTransfer) TestMultiTransaction {
	return TestMultiTransaction{
		MultiTransactionType: MultiTransactionSend,
		FromAddress:          tr.From,
		ToAddress:            tr.To,
		FromToken:            tr.Token,
		ToToken:              tr.Token,
		FromAmount:           tr.Value,
		Timestamp:            tr.Timestamp,
	}
}

func GenerateTestSwapMultiTransaction(tr TestTransfer, toToken string, toAmount int64) TestMultiTransaction {
	return TestMultiTransaction{
		MultiTransactionType: MultiTransactionSwap,
		FromAddress:          tr.From,
		ToAddress:            tr.To,
		FromToken:            tr.Token,
		ToToken:              toToken,
		FromAmount:           tr.Value,
		ToAmount:             toAmount,
		Timestamp:            tr.Timestamp,
	}
}

func GenerateTestBridgeMultiTransaction(fromTr, toTr TestTransfer) TestMultiTransaction {
	return TestMultiTransaction{
		MultiTransactionType: MultiTransactionBridge,
		FromAddress:          fromTr.From,
		ToAddress:            toTr.To,
		FromToken:            fromTr.Token,
		ToToken:              toTr.Token,
		FromAmount:           fromTr.Value,
		ToAmount:             toTr.Value,
		Timestamp:            fromTr.Timestamp,
	}
}

func GenerateTestTransfers(t *testing.T, db *sql.DB, firstStartIndex int, count int) (result []TestTransfer, fromAddresses, toAddresses []eth_common.Address) {
	for i := firstStartIndex; i < (firstStartIndex + count); i++ {
		tr := generateTestTransfer(i)
		fromAddresses = append(fromAddresses, tr.From)
		toAddresses = append(toAddresses, tr.To)
		result = append(result, tr)
	}
	return
}

func InsertTestTransfer(t *testing.T, db *sql.DB, tr *TestTransfer) {
	// Respect `FOREIGN KEY(network_id,address,blk_hash)` of `transfers` table
	tokenType := "eth"
	if tr.Token != "" && strings.ToUpper(tr.Token) != testutils.EthSymbol {
		tokenType = "erc20"
	}
	blkHash := eth_common.HexToHash("4")
	value := sqlite.Int64ToPadded128BitsStr(tr.Value)

	_, err := db.Exec(`
		INSERT OR IGNORE INTO blocks(
			network_id, address, blk_number, blk_hash
		) VALUES (?, ?, ?, ?);
		INSERT INTO transfers (network_id, hash, address, blk_hash, tx,
			sender, receipt, log, type, blk_number, timestamp, loaded,
			multi_transaction_id, base_gas_fee, status, amount_padded128hex
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, 0, ?, ?)`,
		tr.ChainID, tr.To, tr.BlkNumber, blkHash,
		tr.ChainID, tr.Hash, tr.To, blkHash, &JSONBlob{}, tr.From, &JSONBlob{}, &JSONBlob{}, tokenType, tr.BlkNumber, tr.Timestamp, tr.MultiTransactionID, tr.Success, value)
	require.NoError(t, err)
}

func InsertTestPendingTransaction(t *testing.T, db *sql.DB, tr *TestTransfer) {
	_, err := db.Exec(`
		INSERT INTO pending_transactions (network_id, hash, timestamp, from_address, to_address,
			symbol, gas_price, gas_limit, value, data, type, additional_data, multi_transaction_id
		) VALUES (?, ?, ?, ?, ?, 'ETH', 0, 0, ?, '', 'test', '', ?)`,
		tr.ChainID, tr.Hash, tr.Timestamp, tr.From, tr.To, tr.Value, tr.MultiTransactionID)
	require.NoError(t, err)
}

func InsertTestMultiTransaction(t *testing.T, db *sql.DB, tr *TestMultiTransaction) MultiTransactionIDType {
	fromTokenType := tr.FromToken
	if tr.FromToken == "" {
		fromTokenType = testutils.EthSymbol
	}
	toTokenType := tr.ToToken
	if tr.ToToken == "" {
		toTokenType = testutils.EthSymbol
	}
	result, err := db.Exec(`
		INSERT INTO multi_transactions (from_address, from_asset, from_amount, to_address, to_asset, to_amount, type, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tr.FromAddress, fromTokenType, tr.FromAmount, tr.ToAddress, toTokenType, tr.ToAmount, tr.MultiTransactionType, tr.Timestamp)
	require.NoError(t, err)
	rowID, err := result.LastInsertId()
	require.NoError(t, err)
	tr.MultiTransactionID = MultiTransactionIDType(rowID)
	return tr.MultiTransactionID
}
