package alchemy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	sq "github.com/Masterminds/squirrel"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/db/sqlite"
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

func (p *Persistence) GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address common.Address) (*rpc.BlockNumber, *time.Time, error) {
	q := sq.Select("fat.transfer -> '$.blockNum'", "fat.retrieved_at").
		From("fetched_alchemy_transfers fat").
		Where(sq.And{
			sq.Eq{"fat.chain_id": chainID},
			sq.Eq{"fat.address": address},
		}).OrderBy("LENGTH(fat.block_number) DESC, fat.block_number DESC").Limit(1)

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

func (p *Persistence) ClearAll(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM fetched_alchemy_transfers")
	return err
}

// StoredTransfer is one stored transfer as seen from Address. Erc1155 is set for each token of an ERC-1155 transfer.
type StoredTransfer struct {
	ChainID  uint64
	Address  common.Address
	Transfer Transfer
	Erc1155  *Erc1155Metadata
}

// GetHistory returns the stored transfers of the given categories, newest first.
// A transfer stored twice for the same address (from both the outgoing and the incoming query) is returned once,
// and an ERC-1155 transfer gives one entry per token.
func (p *Persistence) GetHistory(ctx context.Context, chainIDs []uint64, addresses []common.Address, categories []TransferCategory, limit, offset int) ([]StoredTransfer, error) {
	if len(chainIDs) == 0 || len(addresses) == 0 || len(categories) == 0 {
		return []StoredTransfer{}, nil
	}

	query := historyQuery(len(chainIDs), len(addresses), len(categories))

	args := make([]interface{}, 0, len(chainIDs)+len(addresses)+len(categories)+2)
	for _, chainID := range chainIDs {
		args = append(args, chainID)
	}
	for _, address := range addresses {
		args = append(args, address)
	}
	for _, category := range categories {
		args = append(args, string(category))
	}
	args = append(args, limit, offset)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]StoredTransfer, 0)
	for rows.Next() {
		var entry StoredTransfer
		var erc1155JSON sql.NullString
		err := rows.Scan(&entry.ChainID, &entry.Address, sqlite.ToJSONBlob(&entry.Transfer), &erc1155JSON)
		if err != nil {
			return nil, err
		}
		if erc1155JSON.Valid {
			var metadata Erc1155Metadata
			if err := json.Unmarshal([]byte(erc1155JSON.String), &metadata); err != nil {
				return nil, err
			}
			entry.Erc1155 = &metadata
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func historyQuery(chainCount, addressCount, categoryCount int) string {
	return `
	SELECT fat.chain_id, fat.address, fat.transfer, m.value
	FROM fetched_alchemy_transfers fat
	LEFT JOIN json_each(fat.transfer, '$.erc1155Metadata') m
	WHERE fat.rowid IN (
		SELECT MIN(rowid)
		FROM fetched_alchemy_transfers
		WHERE chain_id IN (` + placeholders(chainCount) + `)
			AND address IN (` + placeholders(addressCount) + `)
		GROUP BY chain_id, address, COALESCE(NULLIF(json_extract(transfer, '$.uniqueId'), ''), rowid)
	)
		AND json_extract(fat.transfer, '$.category') IN (` + placeholders(categoryCount) + `)
	ORDER BY
		CAST(strftime('%s', json_extract(fat.transfer, '$.metadata.blockTimestamp')) AS INTEGER) DESC,
		LENGTH(fat.block_number) DESC,
		fat.block_number DESC,
		fat.chain_id,
		fat.hash,
		json_extract(fat.transfer, '$.uniqueId'),
		fat.address,
		m.key
	LIMIT ? OFFSET ?`
}

func placeholders(count int) string {
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}
