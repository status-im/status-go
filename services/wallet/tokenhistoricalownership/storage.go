package tokenhistoricalownership

//go:generate go tool mockgen -package=mock_tokenhistoricalownership -destination=mock/storage.go . StorageInterface

import (
	"database/sql"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// TokenOwnership represents a historical token ownership record
type TokenOwnership struct {
	OwnerAddress common.Address
	ChainID      uint64
	TokenAddress common.Address
	Timestamp    int64
}

// StorageInterface defines the interface for historical token ownership storage
type StorageInterface interface {
	MarkAsOwned(ownerAddress common.Address, chainID uint64, tokenAddress common.Address) (bool, error)
	GetOwnedTokens(ownerAddress common.Address) ([]TokenOwnership, error)
	GetOwnedTokensByChain(ownerAddress common.Address, chainID uint64) ([]TokenOwnership, error)
	IsOwned(ownerAddress common.Address, chainID uint64, tokenAddress common.Address) (bool, error)
	RemoveOwnerRecords(ownerAddress common.Address) error
}

// Storage handles persistence of historical token ownership
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage instance
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// MarkAsOwned records a token as owned by an address
// Returns true if this is a new entry, false if it already existed
func (s *Storage) MarkAsOwned(ownerAddress common.Address, chainID uint64, tokenAddress common.Address) (bool, error) {
	timestamp := time.Now().Unix()

	// Check if the record already exists
	var count int
	checkQuery := `
		SELECT COUNT(*) FROM token_historical_ownership
		WHERE owner_address = ? AND chain_id = ? AND token_address = ?
	`
	err := s.db.QueryRow(checkQuery, ownerAddress.Hex(), chainID, tokenAddress.Hex()).Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		// Record exists, update last_detected_timestamp
		updateQuery := `
			UPDATE token_historical_ownership
			SET last_detected_timestamp = ?
			WHERE owner_address = ? AND chain_id = ? AND token_address = ?
		`
		_, err := s.db.Exec(updateQuery, timestamp, ownerAddress.Hex(), chainID, tokenAddress.Hex())
		if err != nil {
			return false, err
		}
		return false, nil // Existing entry
	}

	// Record doesn't exist, insert it
	insertQuery := `
		INSERT INTO token_historical_ownership (owner_address, chain_id, token_address, first_detected_timestamp, last_detected_timestamp)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(insertQuery, ownerAddress.Hex(), chainID, tokenAddress.Hex(), timestamp, timestamp)
	if err != nil {
		return false, err
	}

	return true, nil // New entry
}

// GetOwnedTokens returns all tokens ever owned by an address
func (s *Storage) GetOwnedTokens(ownerAddress common.Address) ([]TokenOwnership, error) {
	query := `
		SELECT owner_address, chain_id, token_address, last_detected_timestamp
		FROM token_historical_ownership
		WHERE owner_address = ?
		ORDER BY last_detected_timestamp DESC
	`

	rows, err := s.db.Query(query, ownerAddress.Hex())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenOwnership
	for rows.Next() {
		var (
			ownerHex string
			chainID  uint64
			tokenHex string
			ts       int64
		)

		if err := rows.Scan(&ownerHex, &chainID, &tokenHex, &ts); err != nil {
			return nil, err
		}

		tokens = append(tokens, TokenOwnership{
			OwnerAddress: common.HexToAddress(ownerHex),
			ChainID:      chainID,
			TokenAddress: common.HexToAddress(tokenHex),
			Timestamp:    ts,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetOwnedTokensByChain returns all tokens ever owned by an address on a specific chain
func (s *Storage) GetOwnedTokensByChain(ownerAddress common.Address, chainID uint64) ([]TokenOwnership, error) {
	query := `
		SELECT owner_address, chain_id, token_address, last_detected_timestamp
		FROM token_historical_ownership
		WHERE owner_address = ? AND chain_id = ?
		ORDER BY last_detected_timestamp DESC
	`

	rows, err := s.db.Query(query, ownerAddress.Hex(), chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenOwnership
	for rows.Next() {
		var (
			ownerHex string
			chainID  uint64
			tokenHex string
			ts       int64
		)

		if err := rows.Scan(&ownerHex, &chainID, &tokenHex, &ts); err != nil {
			return nil, err
		}

		tokens = append(tokens, TokenOwnership{
			OwnerAddress: common.HexToAddress(ownerHex),
			ChainID:      chainID,
			TokenAddress: common.HexToAddress(tokenHex),
			Timestamp:    ts,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// IsOwned checks if a token was ever owned by an address
func (s *Storage) IsOwned(ownerAddress common.Address, chainID uint64, tokenAddress common.Address) (bool, error) {
	query := `
		SELECT COUNT(*) FROM token_historical_ownership
		WHERE owner_address = ? AND chain_id = ? AND token_address = ?
	`

	var count int
	err := s.db.QueryRow(query, ownerAddress.Hex(), chainID, tokenAddress.Hex()).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// RemoveOwnerRecords removes all ownership records for an address (used when account is removed)
func (s *Storage) RemoveOwnerRecords(ownerAddress common.Address) error {
	query := `DELETE FROM token_historical_ownership WHERE owner_address = ?`
	_, err := s.db.Exec(query, ownerAddress.Hex())
	return err
}
