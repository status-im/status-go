package activityfetcher

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	sq "github.com/Masterminds/squirrel"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/sqlite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type PersistenceInterface interface {
	SaveActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, activity thirdparty.ActivityEntryContainer) error
	GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address common.Address) (*rpc.BlockNumber, *time.Time, error)
}

type Persistence struct {
	db *sql.DB
}

func NewPersistence(db *sql.DB) *Persistence {
	return &Persistence{db: db}
}

func (p *Persistence) SaveActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, activity thirdparty.ActivityEntryContainer) (err error) {
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

	return saveActivity(tx, chainID, parameters, activity)
}

func saveActivity(creator sqlite.StatementCreator, chainID uint64, parameters thirdparty.ActivityFetchParameters, activity thirdparty.ActivityEntryContainer) error {
	id := uuid.New().String()

	err := saveFetchParameters(creator, id, chainID, parameters, activity.NextCursor, activity.PreviousCursor, activity.Provider)
	if err != nil {
		return err
	}

	err = saveActivityEntries(creator, id, activity.Items)
	if err != nil {
		return err
	}

	return nil
}

func saveFetchParameters(creator sqlite.StatementCreator, id string, chainID uint64, parameters thirdparty.ActivityFetchParameters, nextCursor, previousCursor, provider string) error {
	q := sq.Insert("fetched_activity_fetch_parameters").
		Columns("id", "chain_id", "parameters", "next_cursor", "previous_cursor", "provider").
		Values(id, chainID, sqlite.ToJSONBlob(parameters), nextCursor, previousCursor, provider)

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

func saveActivityEntries(creator sqlite.StatementCreator, id string, entries []thirdparty.ActivityEntry) error {
	for _, entry := range entries {
		err := saveActivityEntry(creator, id, entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveActivityEntry(creator sqlite.StatementCreator, id string, entry thirdparty.ActivityEntry) error {
	q := sq.Insert("fetched_activity_entries").
		Columns("fetch_parameters_id", "entry").
		Values(id, sqlite.ToJSONBlob(entry))

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

func (p *Persistence) GetActivity(ctx context.Context, chainIDs []uint64, addresses []common.Address, limit uint64) ([]thirdparty.ActivityEntry, error) {
	q := sq.Select("e.entry").
		From("fetched_activity_entries e").
		LeftJoin(`fetched_activity_fetch_parameters fp ON
			e.fetch_parameters_id = fp.id`).
		Where(sq.And{
			sq.Or{
				sq.Eq{"e.chain_id_out": chainIDs},
				sq.Eq{"e.chain_id_in": chainIDs},
			},
			sq.Or{
				sq.Eq{"e.sender": addresses},
				sq.Eq{"e.recipient": addresses},
			},
		}).GroupBy("e.entry")

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

	return rowsToActivityEntries(rows)
}

func rowsToActivityEntries(rows *sql.Rows) ([]thirdparty.ActivityEntry, error) {
	var entries []thirdparty.ActivityEntry
	for rows.Next() {
		var entry thirdparty.ActivityEntry
		var entryJSON = sqlite.ToJSONBlob(&entry)
		err := rows.Scan(entryJSON)
		if err != nil {
			return nil, err
		}
		if !entryJSON.Valid {
			return nil, errors.New("invalid entry")
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (p *Persistence) GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address common.Address) (*rpc.BlockNumber, *time.Time, error) {
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
