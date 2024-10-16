package routeexecution

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/sqlite"
)

type DB struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

func (db *DB) PutRouteData(routeData *RouteData) (err error) {
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

	if err = putRouteInputParams(tx, routeData.RouteInputParams); err != nil {
		return
	}

	if err = putBuildTxParams(tx, routeData.BuildInputParams); err != nil {
		return
	}

	if err = putPathsData(tx, routeData.RouteInputParams.Uuid, routeData.PathsData); err != nil {
		return
	}

	return
}

func (db *DB) GetRouteData(uuid string) (*RouteData, error) {
	return getRouteData(db.db, uuid)
}

func putRouteInputParams(creator sqlite.StatementCreator, p *requests.RouteInputParams) error {
	q := sq.Replace("route_input_parameters").
		SetMap(sq.Eq{"route_input_params_json": &sqlite.JSONBlob{Data: p}})

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
		SetMap(sq.Eq{"route_build_tx_params_json": &sqlite.JSONBlob{Data: p}})

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

func putPathsData(creator sqlite.StatementCreator, uuid string, d []*PathData) error {
	for i, pathData := range d {
		if err := putPathData(creator, uuid, i, pathData); err != nil {
			return err
		}
	}
	return nil
}

func putPathData(creator sqlite.StatementCreator, uuid string, pathIdx int, d *PathData) (err error) {
	err = putPath(creator, uuid, pathIdx, d.Path)
	if err != nil {
		return
	}

	for txIdx, txData := range d.TransactionsData {
		err = putPathTransaction(creator, uuid, pathIdx, txIdx, txData)
		if err != nil {
			return
		}
		err = putSentTransaction(creator, txData)
		if err != nil {
			return
		}
	}

	return
}

func putPath(
	creator sqlite.StatementCreator,
	uuid string,
	pathIdx int,
	p *routes.Path) error {
	q := sq.Replace("route_paths").
		SetMap(sq.Eq{"uuid": uuid, "path_idx": pathIdx, "path_json": &sqlite.JSONBlob{Data: p}})

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
	pathIdx int,
	txIdx int,
	txData *TransactionData,
) error {
	q := sq.Replace("route_path_transactions").
		SetMap(sq.Eq{
			"uuid":         uuid,
			"path_idx":     pathIdx,
			"tx_idx":       txIdx,
			"is_approval":  txData.IsApproval,
			"chain_id":     txData.ChainID,
			"tx_hash":      txData.TxHash[:],
			"tx_args_json": &sqlite.JSONBlob{Data: txData.TxArgs},
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

func putSentTransaction(
	creator sqlite.StatementCreator,
	txData *TransactionData,
) error {
	q := sq.Replace("sent_transactions").
		SetMap(sq.Eq{
			"chain_id": txData.ChainID,
			"tx_hash":  txData.TxHash[:],
			"tx_json":  &sqlite.JSONBlob{Data: txData.Tx},
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

func getRouteData(creator sqlite.StatementCreator, uuid string) (*RouteData, error) {
	routeInputParams, err := getRouteInputParams(creator, uuid)
	if err != nil {
		return nil, err
	}

	buildTxParams, err := getBuildTxParams(creator, uuid)
	if err != nil {
		return nil, err
	}

	pathsData, err := getPathsData(creator, uuid)
	if err != nil {
		return nil, err
	}

	return &RouteData{
		RouteInputParams: routeInputParams,
		BuildInputParams: buildTxParams,
		PathsData:        pathsData,
	}, nil
}

func getRouteInputParams(creator sqlite.StatementCreator, uuid string) (*requests.RouteInputParams, error) {
	var p requests.RouteInputParams
	q := sq.Select("route_input_params_json").
		From("route_input_parameters").
		Where(sq.Eq{"uuid": uuid})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(args...).Scan(&sqlite.JSONBlob{Data: &p})
	return &p, err
}

func getBuildTxParams(creator sqlite.StatementCreator, uuid string) (*requests.RouterBuildTransactionsParams, error) {
	var p requests.RouterBuildTransactionsParams
	q := sq.Select("route_build_tx_params_json").
		From("route_build_tx_parameters").
		Where(sq.Eq{"uuid": uuid})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(args...).Scan(&sqlite.JSONBlob{Data: &p})
	return &p, err
}

func getPathsData(creator sqlite.StatementCreator, uuid string) ([]*PathData, error) {
	var pathsData []*PathData

	paths, err := getPaths(creator, uuid)
	if err != nil {
		return nil, err
	}

	for pathIdx, path := range paths {
		pathData := &PathData{Path: path}
		txs, err := getPathTransactions(creator, uuid, pathIdx)
		if err != nil {
			return nil, err
		}
		pathData.TransactionsData = txs

		pathsData = append(pathsData, pathData)
	}

	return pathsData, nil
}

func getPaths(creator sqlite.StatementCreator, uuid string) ([]*routes.Path, error) {
	var paths []*routes.Path
	q := sq.Select("path_json").
		From("route_paths").
		Where(sq.Eq{"uuid": uuid}).
		OrderBy("path_idx ASC")

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p routes.Path
		err = rows.Scan(&sqlite.JSONBlob{Data: &p})
		if err != nil {
			return nil, err
		}
		paths = append(paths, &p)
	}

	return paths, nil
}

func getPathTransactions(creator sqlite.StatementCreator, uuid string, pathIdx int) ([]*TransactionData, error) {
	txs := make([]*TransactionData, 0, 2)
	q := sq.Select("rpt.is_approval", "rpt.chain_id", "rpt.tx_hash", "rpt.tx_args_json", "st.tx_json").
		From("route_path_transactions rpt").
		LeftJoin(`sent_transactions st ON 
			rpt.chain_id = st.chain_id AND 
			rpt.tx_hash = st.tx_hash`).
		Where(sq.Eq{"rpt.uuid": uuid, "rpt.path_idx": pathIdx}).
		OrderBy("rpt.tx_idx ASC")

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tx TransactionData
		var txHash sql.RawBytes
		err = rows.Scan(&tx.IsApproval, &tx.ChainID, &txHash, &sqlite.JSONBlob{Data: &tx.TxArgs}, &sqlite.JSONBlob{Data: &tx.Tx})
		if err != nil {
			return nil, err
		}
		if len(txHash) > 0 {
			tx.TxHash = types.BytesToHash(txHash)
		}
		txs = append(txs, &tx)
	}

	return txs, nil
}
