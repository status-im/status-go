package alchemy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	sq "github.com/Masterminds/squirrel"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	sqlite2 "github.com/status-im/status-go/internal/db/sqlite"
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

func saveTransfers(creator sqlite2.StatementCreator, transfers []Transfer, chainID uint64, address common.Address) error {
	id := uuid.New().String()

	for _, transfer := range transfers {
		err := saveTransfer(creator, id, transfer, chainID, address)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveTransfer(creator sqlite2.StatementCreator, id string, transfer Transfer, chainID uint64, address common.Address) error {
	q := sq.Insert("fetched_alchemy_transfers").
		Columns("transfer", "chain_id", "address").
		Values(sqlite2.ToJSONBlob(transfer), chainID, address)

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
		var transferJSON = sqlite2.ToJSONBlob(&transfer)
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
	var lastFetchedBlockJSON = sqlite2.ToJSONBlob(&lastFetchedBlock)
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

// TokenInfo represents a token extracted from activity transfers
type TokenInfo struct {
	Address  common.Address
	Category TransferCategory
}

// GetActivityTokens returns unique tokens from activity transfers for a given account and chain
// It extracts token information from the transfer JSON and returns Native and ERC20 tokens only
func (p *Persistence) GetActivityTokens(chainID uint64, address common.Address) ([]TokenInfo, error) {
	// Query to extract category and token address from JSON
	// For external/internal transfers (native), rawContract.address is often empty
	// For ERC20 transfers, rawContract.address contains the token contract address
	query := `
		SELECT DISTINCT
			json_extract(transfer, '$.category') as category,
			json_extract(transfer, '$.rawContract.address') as token_address
		FROM fetched_alchemy_transfers
		WHERE chain_id = ? AND address = ?
		AND json_extract(transfer, '$.category') IN ('external', 'internal', 'erc20')
	`

	rows, err := p.db.Query(query, chainID, address.Hex())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokensMap := make(map[string]TokenInfo)
	for rows.Next() {
		var category string
		var tokenAddress sql.NullString

		if err := rows.Scan(&category, &tokenAddress); err != nil {
			return nil, err
		}

		var info TokenInfo
		info.Category = TransferCategory(category)

		// For native transfers (external/internal), use zero address
		if info.Category == TransferCategoryExternal || info.Category == TransferCategoryInternal {
			info.Address = common.Address{}
		} else if tokenAddress.Valid && tokenAddress.String != "" {
			info.Address = common.HexToAddress(tokenAddress.String)
		} else {
			// Skip if no valid token address for non-native transfers
			continue
		}

		// Use address+category as key to ensure uniqueness
		key := fmt.Sprintf("%s-%s", info.Address.Hex(), info.Category)
		tokensMap[key] = info
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Convert map to slice
	tokens := make([]TokenInfo, 0, len(tokensMap))
	for _, token := range tokensMap {
		tokens = append(tokens, token)
	}

	return tokens, nil
}
