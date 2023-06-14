package collectibles

import (
	"database/sql"
	"fmt"

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
