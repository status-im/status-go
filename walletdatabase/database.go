package walletdatabase

import (
	"database/sql"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/walletdatabase/migrations"
)

const (
	batchSize = 1000
)

type DbInitializer struct {
}

func (a DbInitializer) Initialize(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	return InitializeDB(path, password, kdfIterationsNumber)
}

var walletCustomSteps = []*sqlite.PostStep{
	{Version: 1721166023, CustomMigration: migrateWalletTransactionToAndEventLogAddress, RollBackVersion: 1720206965},
}

func doMigration(db *sql.DB) error {
	// Run all the new migrations
	return migrations.Migrate(db, walletCustomSteps)
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string, kdfIterationsNumber int) (*sql.DB, error) {
	db, err := sqlite.OpenDB(path, password, kdfIterationsNumber)
	if err != nil {
		return nil, err
	}

	err = doMigration(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func migrateWalletTransactionToAndEventLogAddress(sqlTx *sql.Tx) error {
	var batchEntries [][]interface{}

	// Extract Transaction To addresses and Event Log Address and
	// add the information into the new columns
	newColumnsAndIndexSetup := `
		ALTER TABLE transfers ADD COLUMN transaction_to BLOB;
		ALTER TABLE transfers ADD COLUMN event_log_address BLOB;`

	rowIndex := 0
	mightHaveRows := true

	_, err := sqlTx.Exec(newColumnsAndIndexSetup)
	if err != nil {
		return err
	}

	for mightHaveRows {
		var chainID uint64
		var hash common.Hash
		var address common.Address

		rows, err := sqlTx.Query(`SELECT hash, address, network_id, tx, log FROM transfers WHERE tx IS NOT NULL LIMIT ? OFFSET ?`, batchSize, rowIndex)
		if err != nil {
			return err
		}

		curProcessed := 0
		for rows.Next() {
			tx := &types.Transaction{}
			l := &types.Log{}

			// Scan row data into the transaction and log objects
			nullableTx := sqlite.JSONBlob{Data: tx}
			nullableL := sqlite.JSONBlob{Data: l}
			err = rows.Scan(&hash, &address, &chainID, &nullableTx, &nullableL)
			if err != nil {
				if strings.Contains(err.Error(), "missing required field") {
					// Some Arb and Opt transaction types don't contain all required fields
					continue
				}
				rows.Close()
				return err
			}

			var currentRow []interface{}

			var transactionTo *common.Address
			var eventLogAddress *common.Address

			if nullableTx.Valid {
				transactionTo = tx.To()
				currentRow = append(currentRow, transactionTo)
			} else {
				currentRow = append(currentRow, nil)
			}

			if nullableL.Valid {
				eventLogAddress = &l.Address
				currentRow = append(currentRow, eventLogAddress)
			} else {
				currentRow = append(currentRow, nil)
			}

			currentRow = append(currentRow, hash, address, chainID)
			batchEntries = append(batchEntries, currentRow)

			curProcessed++
		}
		rowIndex += curProcessed

		// Check if there was an error in the last rows.Next()
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		mightHaveRows = (curProcessed == batchSize)

		// insert extracted data into the new columns
		if len(batchEntries) > 0 {
			var stmt *sql.Stmt
			stmt, err = sqlTx.Prepare(`UPDATE transfers SET transaction_to = ?, event_log_address = ?
				WHERE hash = ? AND address = ? AND network_id = ?`)
			if err != nil {
				return err
			}

			for _, dataEntry := range batchEntries {
				_, err = stmt.Exec(dataEntry...)
				if err != nil {
					return err
				}
			}

			// Reset placeHolders and batchEntries for the next batch
			batchEntries = [][]interface{}{}
		}
	}

	return nil
}
