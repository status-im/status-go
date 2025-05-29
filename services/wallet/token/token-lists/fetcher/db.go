package fetcher

import "database/sql"

func (t *TokenListsFetcher) StoreTokenList(id string, source string, etag string, jsonData string) error {
	_, err := t.walletDb.Exec(`
	INSERT INTO
		token_lists (id, source, etag, tokens_json)
	VALUES
		(?, ?, ?, ?)`,
		id, source, etag, jsonData)
	return err
}

// GetEtagForTokenList returns the etag for the token list with the given id.
// If the token list does not exist, it returns an empty string.
func (t *TokenListsFetcher) GetEtagForTokenList(id string) (string, error) {
	var etag sql.NullString
	err := t.walletDb.QueryRow("SELECT etag FROM token_lists WHERE id = ?", id).Scan(&etag)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if !etag.Valid {
		return "", nil
	}
	return etag.String, nil
}

func (t *TokenListsFetcher) GetAllTokenLists() ([]FetchedTokenList, error) {
	rows, err := t.walletDb.Query("SELECT id, source, etag, fetched, tokens_json FROM token_lists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokenLists []FetchedTokenList
	for rows.Next() {
		var (
			tokenList FetchedTokenList
			etag      sql.NullString
			fetched   sql.NullTime
		)
		err = rows.Scan(&tokenList.ID, &tokenList.SourceURL, &etag, &fetched, &tokenList.JsonData)
		if err != nil {
			return nil, err
		}
		if etag.Valid {
			tokenList.Etag = etag.String
		}
		if fetched.Valid {
			tokenList.Fetched = fetched.Time
		}
		tokenLists = append(tokenLists, tokenList)
	}

	return tokenLists, nil
}
