package fetcher

import "database/sql"

func (t *TokenListsFetcher) StoreTokenList(id string, jsonData string) error {
	_, err := t.walletDb.Exec(`
	INSERT INTO
		token_lists (id, tokens_json)
	VALUES
		(?, ?)`,
		id, jsonData)
	return err
}

func (t *TokenListsFetcher) GetAllTokenLists() ([]FetchedTokenList, error) {
	rows, err := t.walletDb.Query("SELECT id, fetched, tokens_json FROM token_lists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokenLists []FetchedTokenList
	for rows.Next() {
		var (
			tokenList FetchedTokenList
			fetched   sql.NullTime
		)
		err = rows.Scan(&tokenList.ID, &fetched, &tokenList.JsonData)
		if err != nil {
			return nil, err
		}
		if fetched.Valid {
			tokenList.Fetched = fetched.Time
		}
		tokenLists = append(tokenLists, tokenList)
	}

	return tokenLists, nil
}
