package token

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type storedContent struct {
	SourceURL string
	Etag      string
	Fetched   time.Time
	Data      []byte
}

// contentStore provides content store implementation for storing and retrieving list of token lists and token lists to the database.
type contentStore struct {
	walletDb *sql.DB
}

func NewContentStore(walletDb *sql.DB) *contentStore {
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

func (c *contentStore) Get(id string) (storedContent, error) {
	var (
		content storedContent
		etag    sql.NullString
		fetched sql.NullTime
	)
	err := c.walletDb.QueryRow("SELECT source, etag, fetched, tokens_json FROM token_lists WHERE id = ?", id).
		Scan(&content.SourceURL, &etag, &fetched, &content.Data)
	if err != nil && err != sql.ErrNoRows {
		return storedContent{}, err
	}
	if etag.Valid {
		content.Etag = etag.String
	}
	if fetched.Valid {
		content.Fetched = fetched.Time
	}
	return content, nil
}

func (c *contentStore) Set(id string, content storedContent) error {
	_, err := c.walletDb.Exec(`
	INSERT INTO
		token_lists (id, source, etag, tokens_json)
	VALUES
		(?, ?, ?, ?)`,
		id, content.SourceURL, content.Etag, content.Data)
	return err
}

func (c *contentStore) GetAll() (map[string]storedContent, error) {
	return c.getAll(context.Background())
}

func (c *contentStore) getAll(ctx context.Context) (map[string]storedContent, error) {
	rows, err := c.walletDb.QueryContext(ctx, "SELECT id, source, etag, fetched, tokens_json FROM token_lists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allContents = make(map[string]storedContent)
	for rows.Next() {
		var (
			id      string
			content storedContent
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

	return allContents, rows.Err()
}
