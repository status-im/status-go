package transactions

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/status-im/status-go/sqlite"
)

type TrackedTx struct {
	ID        TxIdentity `json:"id"`
	Timestamp uint64     `json:"timestamp"`
	Status    TxStatus   `json:"status"`
}

type DB struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

func (db *DB) PutTx(transaction TrackedTx) (err error) {
	var tx *sql.Tx
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

	return putTx(tx, transaction)
}

func (db *DB) GetTx(txID TxIdentity) (tx TrackedTx, err error) {
	q := sq.Select("chain_id", "tx_hash", "tx_status", "timestamp").
		From("tracked_transactions").
		Where(sq.Eq{"chain_id": txID.ChainID, "tx_hash": txID.Hash})

	query, args, err := q.ToSql()
	if err != nil {
		return
	}

	row := db.db.QueryRow(query, args...)
	err = row.Scan(&tx.ID.ChainID, &tx.ID.Hash, &tx.Status, &tx.Timestamp)

	return
}

func putTx(creator sqlite.StatementCreator, tx TrackedTx) error {
	q := sq.Replace("tracked_transactions").
		Columns("chain_id", "tx_hash", "tx_status", "timestamp").
		Values(tx.ID.ChainID, tx.ID.Hash, tx.Status, tx.Timestamp)

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)

	return err
}

func (db *DB) UpdateTxStatus(txID TxIdentity, status TxStatus) (err error) {
	var tx *sql.Tx
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

	return updateTxStatus(tx, txID, status)
}

func updateTxStatus(creator sqlite.StatementCreator, txID TxIdentity, status TxStatus) error {
	q := sq.Update("tracked_transactions").
		Set("tx_status", status).
		Where(sq.Eq{"chain_id": txID.ChainID, "tx_hash": txID.Hash})

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)

	return err
}
