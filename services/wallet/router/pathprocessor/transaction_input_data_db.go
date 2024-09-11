package pathprocessor

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/sqlite"
)

type TransactionInputDataDB struct {
	db *sql.DB
}

func NewTransactionInputDataDB(db *sql.DB) *TransactionInputDataDB {
	return &TransactionInputDataDB{
		db: db,
	}
}

func (iDB *TransactionInputDataDB) UpsertInputData(chainID w_common.ChainID, txHash types.Hash, inputData TransactionInputData) error {
	return upsertInputData(iDB.db, chainID, txHash, inputData)
}

func (iDB *TransactionInputDataDB) ReadInputData(chainID w_common.ChainID, txHash types.Hash) (*TransactionInputData, error) {
	return readInputData(iDB.db, chainID, txHash)
}

func upsertInputData(creator sqlite.StatementCreator, chainID w_common.ChainID, txHash types.Hash, inputData TransactionInputData) error {
	q := sq.Replace("transaction_input_data").
		SetMap(sq.Eq{
			"chain_id":         chainID,
			"tx_hash":          txHash.Bytes(),
			"processor_name":   inputData.ProcessorName,
			"from_asset":       inputData.FromAsset,
			"from_amount":      (*bigint.SQLBigIntBytes)(inputData.FromAmount),
			"to_asset":         inputData.ToAsset,
			"to_amount":        (*bigint.SQLBigIntBytes)(inputData.ToAmount),
			"side":             inputData.Side,
			"slippage_bps":     inputData.SlippageBps,
			"approval_amount":  (*bigint.SQLBigIntBytes)(inputData.ApprovalAmount),
			"approval_spender": inputData.ApprovalSpender,
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

func readInputData(creator sqlite.StatementCreator, chainID w_common.ChainID, txHash types.Hash) (*TransactionInputData, error) {
	q := sq.Select(
		"processor_name",
		"from_asset",
		"from_amount",
		"to_asset",
		"to_amount",
		"side",
		"slippage_bps",
		"approval_amount",
		"approval_spender",
	).
		From("transaction_input_data").
		Where(sq.Eq{"chain_id": chainID, "tx_hash": txHash.Bytes()})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	inputData := NewInputData()
	err = stmt.QueryRow(args...).Scan(
		&inputData.ProcessorName,
		&inputData.FromAsset,
		(*bigint.SQLBigIntBytes)(inputData.FromAmount),
		&inputData.ToAsset,
		(*bigint.SQLBigIntBytes)(inputData.ToAmount),
		&inputData.Side,
		&inputData.SlippageBps,
		(*bigint.SQLBigIntBytes)(inputData.ApprovalAmount),
		&inputData.ApprovalSpender,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return inputData, nil
}
