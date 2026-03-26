package communitytokensdatabase

import (
	"database/sql"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Database struct {
	db *sql.DB
}

func (db *Database) logger() *zap.Logger {
	return logutils.ZapLogger().Named("communitytokensdatabase")
}

func (db *Database) dumpAddressesForChain(chainID uint64) []string {
	rows, err := db.db.Query(`SELECT address FROM community_tokens WHERE chain_id=? LIMIT 100`, chainID)
	if err != nil {
		db.logger().Warn("failed to dump community token addresses for chain",
			zap.Uint64("chainID", chainID),
			zap.Error(err),
		)
		return nil
	}
	defer rows.Close()

	addresses := make([]string, 0)
	for rows.Next() {
		var address string
		if scanErr := rows.Scan(&address); scanErr != nil {
			db.logger().Warn("failed to scan community token address while dumping",
				zap.Uint64("chainID", chainID),
				zap.Error(scanErr),
			)
			continue
		}
		addresses = append(addresses, address)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		db.logger().Warn("rows error while dumping community token addresses",
			zap.Uint64("chainID", chainID),
			zap.Error(rowsErr),
		)
	}

	return addresses
}

func NewCommunityTokensDatabase(db *sql.DB) *Database {
	return &Database{db: db}
}

func (db *Database) GetTokenType(chainID uint64, contractAddress string) (protobuf.CommunityTokenType, error) {
	var result = protobuf.CommunityTokenType_UNKNOWN_TOKEN_TYPE
	rows, err := db.db.Query(`SELECT type FROM community_tokens WHERE chain_id=? AND address=? LIMIT 1`, chainID, contractAddress)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&result)
		return result, err
	}

	var caseInsensitiveAddress string
	ciErr := db.db.QueryRow(`SELECT address FROM community_tokens WHERE chain_id=? AND lower(address)=lower(?) LIMIT 1`, chainID, contractAddress).
		Scan(&caseInsensitiveAddress)
	if ciErr == nil {
		db.logger().Warn("community token type lookup failed by exact address but matched case-insensitively",
			zap.Uint64("chainID", chainID),
			zap.String("requestedAddress", contractAddress),
			zap.String("matchedAddress", caseInsensitiveAddress),
		)
	}

	addresses := db.dumpAddressesForChain(chainID)
	db.logger().Warn("community token type not found",
		zap.Uint64("chainID", chainID),
		zap.String("requestedAddress", contractAddress),
		zap.String("requestedAddressLower", strings.ToLower(contractAddress)),
		zap.Int("knownAddressCount", len(addresses)),
		zap.Strings("knownAddresses", addresses),
	)

	return result, fmt.Errorf("can't find token: chainId %v, contractAddress %v", chainID, contractAddress)
}

func (db *Database) GetTokenPrivilegesLevel(chainID uint64, contractAddress string) (token.PrivilegesLevel, error) {
	var result = token.CommunityLevel
	rows, err := db.db.Query(`SELECT privileges_level FROM community_tokens WHERE chain_id=? AND address=? LIMIT 1`, chainID, contractAddress)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&result)
		return result, err
	}

	var caseInsensitiveAddress string
	ciErr := db.db.QueryRow(`SELECT address FROM community_tokens WHERE chain_id=? AND lower(address)=lower(?) LIMIT 1`, chainID, contractAddress).
		Scan(&caseInsensitiveAddress)
	if ciErr == nil {
		db.logger().Warn("community token privilege lookup failed by exact address but matched case-insensitively",
			zap.Uint64("chainID", chainID),
			zap.String("requestedAddress", contractAddress),
			zap.String("matchedAddress", caseInsensitiveAddress),
		)
	}

	addresses := db.dumpAddressesForChain(chainID)
	db.logger().Warn("community token privileges level not found",
		zap.Uint64("chainID", chainID),
		zap.String("requestedAddress", contractAddress),
		zap.String("requestedAddressLower", strings.ToLower(contractAddress)),
		zap.Int("knownAddressCount", len(addresses)),
		zap.Strings("knownAddresses", addresses),
	)

	return result, fmt.Errorf("can't find privileges level: chainId %v, contractAddress %v", chainID, contractAddress)
}

func (db *Database) GetTokens() ([]*token.CommunityToken, error) {
	rows, err := db.db.Query(`SELECT community_id, address, name, symbol, chain_id, decimals, type FROM community_tokens`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*token.CommunityToken
	for rows.Next() {
		token := token.CommunityToken{}
		err := rows.Scan(&token.CommunityID, &token.Address, &token.Name, &token.Symbol, &token.ChainID, &token.Decimals, &token.TokenType)
		if err != nil {
			return nil, err
		}
		result = append(result, &token)
	}
	return result, rows.Err()
}
