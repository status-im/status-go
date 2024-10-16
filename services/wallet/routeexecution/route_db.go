package routeexecution

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/transactions"
)

type DB struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

func (db *DB) PutRouteData(
	routeInputParams requests.RouteInputParams,
	buildInputParams *requests.RouterBuildTransactionsParams,
	transactionDetails []*transfer.RouterTransactionDetails) (err error) {
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

	if err = putRouteInputParams(tx, routeInputParams); err != nil {
		return
	}

	if err = putBuildTxParams(tx, buildInputParams); err != nil {
		return
	}

	if err = putPathsTransactionDetails(tx, routeInputParams.Uuid, transactionDetails); err != nil {
		return
	}

	return
}

func putRouteInputParams(creator sqlite.StatementCreator, p requests.RouteInputParams) error {
	q := sq.Replace("route_input_parameters").
		SetMap(sq.Eq{"route_input_params_json": sqlite.JSONBlob{Data: p}})

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

func putBuildTxParams(creator sqlite.StatementCreator, p *requests.RouterBuildTransactionsParams) error {
	q := sq.Replace("route_build_tx_parameters").
		SetMap(sq.Eq{"route_build_tx_params_json": sqlite.JSONBlob{Data: p}})

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

func putPathsTransactionDetails(creator sqlite.StatementCreator, uuid string, d []*transfer.RouterTransactionDetails) error {
	for i, details := range d {
		if err := putPathTransactionDetails(creator, uuid, i, details); err != nil {
			return err
		}
	}
	return nil
}

func putPathTransactionDetails(creator sqlite.StatementCreator, uuid string, idx int, d *transfer.RouterTransactionDetails) (err error) {
	err = putPath(creator, uuid, idx, d.RouterPath)
	if err != nil {
		return
	}

	chainID := d.RouterPath.FromChain.ChainID

	if d.IsApprovalPlaced() {
		err = putPathTransaction(creator, uuid, idx, true, chainID, d.ApprovalTxSentHash, d.ApprovalTxArgs, d.ApprovalTx)
		if err != nil {
			return
		}
	}

	err = putPathTransaction(creator, uuid, idx, false, chainID, d.TxSentHash, d.TxArgs, d.Tx)

	return
}

func putPath(
	creator sqlite.StatementCreator,
	uuid string,
	idx int,
	p *routes.Path) error {
	q := sq.Replace("route_paths").
		SetMap(sq.Eq{"uuid": uuid, "idx": idx, "path_json": sqlite.JSONBlob{Data: p}})

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

func putPathTransaction(
	creator sqlite.StatementCreator,
	uuid string,
	idx int,
	is_approval bool,
	chainID uint64,
	txHash types.Hash,
	txArgs *transactions.SendTxArgs,
	tx *ethTypes.Transaction,
) error {
	q := sq.Replace("route_path_transactions").
		SetMap(sq.Eq{
			"uuid":         uuid,
			"idx":          idx,
			"is_approval":  is_approval,
			"chain_id":     chainID,
			"tx_hash":      txHash,
			"tx_args_json": sqlite.JSONBlob{Data: txArgs},
			"tx_json":      sqlite.JSONBlob{Data: tx},
		})

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
