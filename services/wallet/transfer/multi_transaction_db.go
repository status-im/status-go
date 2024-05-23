package transfer

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
)

// DO NOT CREATE IT MANUALLY! Use NewMultiTxDetails() instead
type MultiTxDetails struct {
	AnyAddress  common.Address
	FromAddress common.Address
	ToAddress   common.Address
	ToChainID   uint64
	CrossTxID   string
	Type        MultiTransactionType
}

func NewMultiTxDetails() *MultiTxDetails {
	details := &MultiTxDetails{}
	details.Type = MultiTransactionTypeInvalid
	return details
}

type MultiTransactionDB struct {
	db *sql.DB
}

func NewMultiTransactionDB(db *sql.DB) *MultiTransactionDB {
	return &MultiTransactionDB{
		db: db,
	}
}

func (mtDB *MultiTransactionDB) CreateMultiTransaction(multiTransaction *MultiTransaction) error {
	insert, err := mtDB.db.Prepare(fmt.Sprintf(`INSERT INTO multi_transactions (%s)
											VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, multiTransactionColumns))
	if err != nil {
		return err
	}
	_, err = insert.Exec(
		multiTransaction.ID,
		multiTransaction.FromNetworkID,
		multiTransaction.FromTxHash,
		multiTransaction.FromAddress,
		multiTransaction.FromAsset,
		multiTransaction.FromAmount.String(),
		multiTransaction.ToNetworkID,
		multiTransaction.ToTxHash,
		multiTransaction.ToAddress,
		multiTransaction.ToAsset,
		multiTransaction.ToAmount.String(),
		multiTransaction.Type,
		multiTransaction.CrossTxID,
		multiTransaction.Timestamp,
	)
	if err != nil {
		return err
	}
	defer insert.Close()

	return err
}

func (mtDB *MultiTransactionDB) ReadMultiTransactions(ids []wallet_common.MultiTransactionIDType) ([]*MultiTransaction, error) {
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, v := range ids {
		placeholders[i] = "?"
		args[i] = v
	}

	stmt, err := mtDB.db.Prepare(fmt.Sprintf(`SELECT %s
											FROM multi_transactions
											WHERE id in (%s)`,
		selectMultiTransactionColumns,
		strings.Join(placeholders, ",")))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToMultiTransactions(rows)
}

func (mtDB *MultiTransactionDB) ReadMultiTransactionsByDetails(details *MultiTxDetails) ([]*MultiTransaction, error) {
	if details == nil {
		return nil, fmt.Errorf("details is nil")
	}

	whereClause := ""

	args := []interface{}{}

	if (details.AnyAddress != common.Address{}) {
		whereClause += "(from_address=? OR to_address=?) AND "
		args = append(args, details.AnyAddress, details.AnyAddress)
	}
	if (details.FromAddress != common.Address{}) {
		whereClause += "from_address=? AND "
		args = append(args, details.FromAddress)
	}
	if (details.ToAddress != common.Address{}) {
		whereClause += "to_address=? AND "
		args = append(args, details.ToAddress)
	}
	if details.ToChainID != 0 {
		whereClause += "to_network_id=? AND "
		args = append(args, details.ToChainID)
	}
	if details.CrossTxID != "" {
		whereClause += "cross_tx_id=? AND "
		args = append(args, details.CrossTxID)
	}
	if details.Type != MultiTransactionTypeInvalid {
		whereClause += "type=? AND "
		args = append(args, details.Type)
	}

	stmt, err := mtDB.db.Prepare(fmt.Sprintf(`SELECT %s
											FROM multi_transactions
											WHERE %s`,
		selectMultiTransactionColumns, whereClause[:len(whereClause)-5]))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToMultiTransactions(rows)
}

func (mtDB *MultiTransactionDB) UpdateMultiTransaction(multiTransaction *MultiTransaction) error {
	if multiTransaction.ID == wallet_common.NoMultiTransactionID {
		return fmt.Errorf("no multitransaction ID")
	}

	update, err := mtDB.db.Prepare(fmt.Sprintf(`REPLACE INTO multi_transactions (%s)
	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, multiTransactionColumns))

	if err != nil {
		return err
	}
	_, err = update.Exec(
		multiTransaction.ID,
		multiTransaction.FromNetworkID,
		multiTransaction.FromTxHash,
		multiTransaction.FromAddress,
		multiTransaction.FromAsset,
		multiTransaction.FromAmount.String(),
		multiTransaction.ToNetworkID,
		multiTransaction.ToTxHash,
		multiTransaction.ToAddress,
		multiTransaction.ToAsset,
		multiTransaction.ToAmount.String(),
		multiTransaction.Type,
		multiTransaction.CrossTxID,
		multiTransaction.Timestamp,
	)
	if err != nil {
		return err
	}
	return update.Close()
}

func (mtDB *MultiTransactionDB) DeleteMultiTransaction(id wallet_common.MultiTransactionIDType) error {
	_, err := mtDB.db.Exec(`DELETE FROM multi_transactions WHERE id=?`, id)
	return err
}
