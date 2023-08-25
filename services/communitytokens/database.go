package communitytokens

import (
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Database struct {
	db *sql.DB
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
	return result, fmt.Errorf("can't find privileges level: chainId %v, contractAddress %v", chainID, contractAddress)
}
