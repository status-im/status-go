package walletdatabase

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/common/dbsetup"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase/migrations"
)

const (
	erc20ReceiptTestDataTemplate = `{"type":"0x2","root":"0x","status":"0x%d","cumulativeGasUsed":"0x10f8d2c","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000001008000000000000000000000000000000000000002000000000020000000000000000000800000000000000000000000010000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000800000000000000000000","logs":[{"address":"0x98339d8c260052b7ad81c28c16c0b98420f2b46a","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000000000000000000000000000000000000000000000","0x000000000000000000000000e2d622c817878da5143bbe06866ca8e35273ba8a"],"data":"0x0000000000000000000000000000000000000000000000000000000000989680","blockNumber":"0x825527","transactionHash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a","transactionIndex":"0x6c","blockHash":"0x69e0f829a557052c134cd7e21c220507d91bc35c316d3c47217e9bd362270274","logIndex":"0xcd","removed":false}],"transactionHash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x8623","blockHash":"0x69e0f829a557052c134cd7e21c220507d91bc35c316d3c47217e9bd362270274","blockNumber":"0x825527","transactionIndex":"0x6c"}`
	erc20TxTestData              = `{"type":"0x2","nonce":"0x3d","gasPrice":"0x0","maxPriorityFeePerGas":"0x8c347c90","maxFeePerGas":"0x45964d43a4","gas":"0x8623","value":"0x0","input":"0x40c10f19000000000000000000000000e2d622c817878da5143bbe06866ca8e35273ba8a0000000000000000000000000000000000000000000000000000000000989680","v":"0x0","r":"0xbcac4bb290d48b467bb18ac67e98050b5f316d2c66b2f75dcc1d63a45c905d21","s":"0x10c15517ea9cabd7fe134b270daabf5d2e8335e935d3e021f54a4efaffb37cd2","to":"0x98339d8c260052b7ad81c28c16c0b98420f2b46a","chainId":"0x5","accessList":[],"hash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a"}`
	erc20LogTestData             = `{"address":"0x98339d8c260052b7ad81c28c16c0b98420f2b46a","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000000000000000000000000000000000000000000000","0x000000000000000000000000e2d622c817878da5143bbe06866ca8e35273ba8a"],"data":"0x0000000000000000000000000000000000000000000000000000000000989680","blockNumber":"0x825527","transactionHash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a","transactionIndex":"0x6c","blockHash":"0x69e0f829a557052c134cd7e21c220507d91bc35c316d3c47217e9bd362270274","logIndex":"0xcd","removed":false}`

	ethReceiptTestData = `{
		"type": "0x2",
		"root": "0x",
		"status": "0x1",
		"cumulativeGasUsed": "0x2b461",
		"logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"logs": [],
		"transactionHash": "0x4ac700ee2a1702f82b3cfdc88fd4d91f767b87fea9b929bd6223c6471a5e05b4",
		"contractAddress": "0x0000000000000000000000000000000000000000",
		"gasUsed": "0x5208",
		"blockHash": "0x25fe164361c1cb4ed1b46996f7b5236d3118144529b31fca037fcda1d8ee684d",
		"blockNumber": "0x5e3294",
		"transactionIndex": "0x3"
	}`
	ethTxTestData = `{
		"type": "0x2",
		"nonce": "0x1",
		"gasPrice": "0x0",
		"maxPriorityFeePerGas": "0x33",
		"maxFeePerGas": "0x3b9aca00",
		"gas": "0x55f0",
		"value": "0x%s",
		"input": "0x",
		"v": "0x0",
		"r": "0xacc277ce156382d6f333cc8d75a56250778b17f1c6d1676af63cf68d53713986",
		"s": "0x32417261484e9796390abb8db13f993965d917836be5cd96df25b9b581de91ec",
		"to": "0xbd54a96c0ae19a220c8e1234f54c940dfab34639",
		"chainId": "0x1a4",
		"accessList": [],
		"hash": "0x4ac700ee2a1702f82b3cfdc88fd4d91f767b87fea9b929bd6223c6471a5e05b4"
	}`
)

func TestMigrateWalletTransactionToAndEventLogAddress(t *testing.T) {
	openDB := func() (*sql.DB, error) {
		return sqlite.OpenDB(sqlite.InMemoryPath, "1234567890", dbsetup.ReducedKDFIterationsNumber)
	}
	db, err := openDB()
	require.NoError(t, err)

	// Migrate until before custom step
	err = migrations.MigrateTo(db, walletCustomSteps, 1720206965)
	require.NoError(t, err)

	// Validate that transfers table has no status column
	exists, err := helpers.ColumnExists(db, "transfers", "transaction_to")
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = helpers.ColumnExists(db, "transfers", "event_log_address")
	require.NoError(t, err)
	require.False(t, exists)

	insertTestTransaction := func(index int, txBlob string, receiptBlob string, logBlob string, ethType bool) error {
		indexStr := strconv.Itoa(index)
		senderStr := strconv.Itoa(index + 1)

		var txValue *string
		if txBlob != "" {
			txValue = &txBlob
		}
		var receiptValue *string
		if receiptBlob != "" {
			receiptValue = &receiptBlob
		}
		var logValue *string
		if logBlob != "" {
			logValue = &logBlob
		}
		entryType := "eth"
		if !ethType {
			entryType = "erc20"
		}
		_, err = db.Exec(`INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash) VALUES (?, ?, ?, ?);
			INSERT INTO transfers (hash, address, sender, network_id, tx, receipt, log, blk_hash, type,  blk_number, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			index, common.HexToAddress(indexStr), index, common.HexToHash(indexStr),
			common.HexToHash(indexStr), common.HexToAddress(indexStr), common.HexToAddress(senderStr), index, txValue, receiptValue, logValue, common.HexToHash(indexStr), entryType, index, index)
		return err
	}

	// Empty transaction, found the usecase in the test DB
	erc20FailReceiptJSON := fmt.Sprintf(erc20ReceiptTestDataTemplate, 0)
	erc20SuccessReceiptJSON := fmt.Sprintf(erc20ReceiptTestDataTemplate, 1)
	err = insertTestTransaction(2, erc20TxTestData, erc20FailReceiptJSON, erc20LogTestData, false)
	require.NoError(t, err)

	err = insertTestTransaction(3, erc20TxTestData, erc20SuccessReceiptJSON, erc20LogTestData, false)
	require.NoError(t, err)

	ethZeroValueTxTestData := fmt.Sprintf(ethTxTestData, "0")
	ethVeryBigValueTxTestData := fmt.Sprintf(ethTxTestData, "12345678901234567890")
	ethOriginalTxTestData := fmt.Sprintf(ethTxTestData, "2386f26fc10000")

	err = insertTestTransaction(4, ethZeroValueTxTestData, ethReceiptTestData, "", true)
	require.NoError(t, err)
	err = insertTestTransaction(5, ethVeryBigValueTxTestData, "", "", true)
	require.NoError(t, err)
	err = insertTestTransaction(6, ethOriginalTxTestData, ethReceiptTestData, "", true)
	require.NoError(t, err)

	failMigrationSteps := []*sqlite.PostStep{
		{
			Version: walletCustomSteps[0].Version,
			CustomMigration: func(sqlTx *sql.Tx) error {
				return errors.New("failed to run custom migration")
			},
			RollBackVersion: walletCustomSteps[0].RollBackVersion,
		},
	}

	// Attempt to run test migration 1721166023 and fail in custom step
	err = migrations.MigrateTo(db, failMigrationSteps, walletCustomSteps[0].Version)
	require.Error(t, err)

	exists, err = helpers.ColumnExists(db, "transfers", "transaction_to")
	require.NoError(t, err)
	require.False(t, exists)

	// Run test migration 1721166023_extract_tx_to_and_event_log_address.<up/down>.sql
	err = migrations.MigrateTo(db, walletCustomSteps, walletCustomSteps[0].Version)
	require.NoError(t, err)

	// Validate that the migration was run and transfers table has now new columns
	exists, err = helpers.ColumnExists(db, "transfers", "transaction_to")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = helpers.ColumnExists(db, "transfers", "event_log_address")
	require.NoError(t, err)
	require.True(t, exists)

	var (
		transactionTo   sql.RawBytes
		eventLogAddress sql.RawBytes
	)

	rows, err := db.Query(`SELECT transaction_to, event_log_address
		FROM transfers ORDER BY timestamp ASC`)
	require.NoError(t, err)

	scanNextData := func() error {
		rows.Next()
		if rows.Err() != nil {
			return rows.Err()
		}
		err := rows.Scan(&transactionTo, &eventLogAddress)
		if err != nil {
			return err
		}
		return nil
	}

	validateTransaction := func(tt *types.Transaction, expectedEntryType w_common.Type, tl *types.Log) {
		if tt == nil {
			require.Empty(t, transactionTo)
			require.Empty(t, eventLogAddress)
		} else {
			if expectedEntryType == w_common.EthTransfer {
				require.NotEmpty(t, transactionTo)
				parsedTransactionTo := common.BytesToAddress(transactionTo)
				require.Equal(t, *tt.To(), parsedTransactionTo)
			} else {
				require.NotEmpty(t, transactionTo)
				parsedTransactionTo := common.BytesToAddress(transactionTo)
				require.Equal(t, *tt.To(), parsedTransactionTo)

				require.NotEmpty(t, eventLogAddress)
				parsedEventLogAddress := common.BytesToAddress(eventLogAddress)
				require.Equal(t, tl.Address, parsedEventLogAddress)
			}
		}
	}

	var successReceipt types.Receipt
	err = json.Unmarshal([]byte(erc20SuccessReceiptJSON), &successReceipt)
	require.NoError(t, err)

	var failReceipt types.Receipt
	err = json.Unmarshal([]byte(erc20FailReceiptJSON), &failReceipt)
	require.NoError(t, err)

	var erc20Log types.Log
	err = json.Unmarshal([]byte(erc20LogTestData), &erc20Log)
	require.NoError(t, err)

	var erc20Tx types.Transaction
	err = json.Unmarshal([]byte(erc20TxTestData), &erc20Tx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&erc20Tx, w_common.Erc20Transfer, &erc20Log)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&erc20Tx, w_common.Erc20Transfer, &erc20Log)

	var zeroTestTx types.Transaction
	err = json.Unmarshal([]byte(ethZeroValueTxTestData), &zeroTestTx)
	require.NoError(t, err)

	var ethReceipt types.Receipt
	err = json.Unmarshal([]byte(ethReceiptTestData), &ethReceipt)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&zeroTestTx, w_common.EthTransfer, nil)

	var bigTestTx types.Transaction
	err = json.Unmarshal([]byte(ethVeryBigValueTxTestData), &bigTestTx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&bigTestTx, w_common.EthTransfer, nil)

	var ethOriginalTestTx types.Transaction
	err = json.Unmarshal([]byte(ethOriginalTxTestData), &ethOriginalTestTx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&ethOriginalTestTx, w_common.EthTransfer, nil)

	err = scanNextData()
	// Validate that we processed all data (no more rows expected)
	require.Error(t, err)

	db.Close()
}
