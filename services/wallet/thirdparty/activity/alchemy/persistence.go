package alchemy

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	sq "github.com/Masterminds/squirrel"

	"github.com/status-im/status-go/sqlite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type Persistence struct {
	db *sql.DB
}

func NewPersistence(db *sql.DB) *Persistence {
	return &Persistence{db: db}
}

func (p *Persistence) SaveTransfers(tt []Transfer, chainID uint64, address common.Address) (err error) {
	var tx *sql.Tx
	tx, err = p.db.Begin()
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

	return saveTransfers(tx, tt, chainID, address)
}

func saveTransfers(creator sqlite.StatementCreator, transfers []Transfer, chainID uint64, address common.Address) error {
	id := uuid.New().String()

	for _, transfer := range transfers {
		err := saveTransfer(creator, id, transfer, chainID, address)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveTransfer(creator sqlite.StatementCreator, id string, transfer Transfer, chainID uint64, address common.Address) error {
	q := sq.Insert("fetched_alchemy_transfers").
		Columns("transfer", "chain_id", "address").
		Values(sqlite.ToJSONBlob(transfer), chainID, address)

	query, args, err := q.ToSql()
	if err != nil {
		return err
	}

	stmt, err := creator.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(args...)
	return err
}

func (p *Persistence) GetTransfers(chainIDs []uint64, addresses []common.Address, limit uint64) ([]Transfer, error) {
	q := sq.Select("e.transfer").
		From("fetched_alchemy_transfers e").
		Where(sq.And{
			sq.Eq{"e.chain_id": chainIDs},
			sq.Eq{"e.address": addresses}})

	if limit > 0 {
		q = q.Limit(limit)
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := p.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToTransfers(rows)
}

func rowsToTransfers(rows *sql.Rows) ([]Transfer, error) {
	var transfers []Transfer
	for rows.Next() {
		var transfer Transfer
		var transferJSON = sqlite.ToJSONBlob(&transfer)
		err := rows.Scan(transferJSON)
		fmt.Println("Scanned json:", transferJSON)
		if err != nil {
			return nil, err
		}
		if !transferJSON.Valid {
			return nil, errors.New("invalid entry")
		}
		transfers = append(transfers, transfer)
	}
	return transfers, nil
}

func (p *Persistence) GetLastFetchedBlockAndTimestamp(chainID uint64, address common.Address) (*rpc.BlockNumber, *time.Time, error) {
	q := sq.Select("fp.parameters -> '$.toBlock'", "fp.created_at").
		From("fetched_activity_fetch_parameters fp").
		Where(sq.And{
			sq.Eq{"fp.chain_id": chainID},
			sq.Eq{"fp.address": address},
		}).OrderBy("fp.created_at DESC").Limit(1)

	query, args, err := q.ToSql()
	if err != nil {
		return nil, nil, err
	}

	stmt, err := p.db.Prepare(query)
	if err != nil {
		return nil, nil, err
	}
	defer stmt.Close()

	var lastFetchedTimestamp time.Time
	var lastFetchedBlock rpc.BlockNumber
	var lastFetchedBlockJSON = sqlite.ToJSONBlob(&lastFetchedBlock)
	err = stmt.QueryRow(args...).Scan(lastFetchedBlockJSON, &lastFetchedTimestamp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	return &lastFetchedBlock, &lastFetchedTimestamp, nil
}
