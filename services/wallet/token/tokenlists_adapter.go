package token

import (
	"database/sql"
	"fmt"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/autofetcher"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/manager"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
)

// contentStore provides content store implementation for storing and retrieving list of token lists and token lists to the database.
type contentStore struct {
	walletDb *sql.DB
}

func NewContentStore(walletDb *sql.DB) autofetcher.ContentStore {
	return &contentStore{walletDb: walletDb}
}

func (c *contentStore) GetEtag(id string) (string, error) {
	var etag sql.NullString
	err := c.walletDb.QueryRow("SELECT etag FROM token_lists WHERE id = ?", id).Scan(&etag)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if !etag.Valid {
		return "", fmt.Errorf("stored etag is invalid")
	}
	return etag.String, nil
}

func (c *contentStore) Get(id string) (autofetcher.Content, error) {
	var (
		content autofetcher.Content
		etag    sql.NullString
		fetched sql.NullTime
	)
	err := c.walletDb.QueryRow("SELECT source, etag, fetched, tokens_json FROM token_lists WHERE id = ?", id).
		Scan(&content.SourceURL, &etag, &fetched, &content.Data)
	if err != nil && err != sql.ErrNoRows {
		return autofetcher.Content{}, err
	}
	if etag.Valid {
		content.Etag = etag.String
	}
	return content, nil
}

func (c *contentStore) Set(id string, content autofetcher.Content) error {
	_, err := c.walletDb.Exec(`
	INSERT INTO
		token_lists (id, source, etag, tokens_json)
	VALUES
		(?, ?, ?, ?)`,
		id, content.SourceURL, content.Etag, content.Data)
	return err
}

func (c *contentStore) GetAll() (map[string]autofetcher.Content, error) {
	rows, err := c.walletDb.Query("SELECT id, source, etag, fetched, tokens_json FROM token_lists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allContents = make(map[string]autofetcher.Content)
	for rows.Next() {
		var (
			id      string
			content autofetcher.Content
			etag    sql.NullString
			fetched sql.NullTime
		)
		err = rows.Scan(&id, &content.SourceURL, &etag, &fetched, &content.Data)
		if err != nil {
			return nil, err
		}
		if etag.Valid {
			content.Etag = etag.String
		}
		if fetched.Valid {
			content.Fetched = fetched.Time
		}
		allContents[id] = content
	}

	return allContents, nil
}

// customTokenStore provides custom token store implementation for retrieving custom tokens from the database.
type customTokenStore struct {
	manager *Manager // this is the esaist way to align with the wallet sdk interface, until we refactor Manager
}

func NewCustomTokenStore(manager *Manager) manager.CustomTokenStore {
	return &customTokenStore{manager: manager}
}

func (c *customTokenStore) GetAll() ([]*types.Token, error) {
	if c.manager == nil {
		return nil, fmt.Errorf("manager is not set")
	}

	customTokens, err := c.manager.getDiscoveredTokens(false)
	if err != nil {
		return nil, err
	}

	tokens := make([]*types.Token, 0)
	for _, token := range customTokens {
		tokens = append(tokens, token.Token)
	}

	return tokens, nil
}
