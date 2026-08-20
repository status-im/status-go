package preferences

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

var (
	errEmptyCategory  = errors.New("category must not be empty")
	errEmptyKey       = errors.New("key must not be empty")
	errEmptyValidKeys = errors.New("validKeys must not be empty; use DeleteCategory to remove all keys in a category")
)

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// Store is the SQLite adapter for PreferenceStore.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func validateCategory(category string) error {
	if category == "" {
		return errEmptyCategory
	}
	return nil
}

func validateCategoryKey(category, key string) error {
	if err := validateCategory(category); err != nil {
		return err
	}
	if key == "" {
		return errEmptyKey
	}
	return nil
}

func upsert(exec sqlExecer, category, key, value string, updatedAt int64) error {
	query, args, err := sq.Insert("preferences").
		Columns("category", "key", "value", "updated_at").
		Values(category, key, value, updatedAt).
		Suffix(`ON CONFLICT (category, key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}

	_, err = exec.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("upsert preference: %w", err)
	}
	return nil
}

func (s *Store) Set(category, key, value string) error {
	if err := validateCategoryKey(category, key); err != nil {
		return err
	}
	return upsert(s.db, category, key, value, time.Now().UnixMilli())
}

func (s *Store) SetMany(category string, kvs map[string]string) (err error) {
	if err = validateCategory(category); err != nil {
		return err
	}
	if len(kvs) == 0 {
		return nil
	}

	var tx *sql.Tx
	tx, err = s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	now := time.Now().UnixMilli()
	for key, value := range kvs {
		if key == "" {
			return errEmptyKey
		}
		if err = upsert(tx, category, key, value, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) Get(category, key string) (value string, found bool, err error) {
	if err := validateCategoryKey(category, key); err != nil {
		return "", false, err
	}

	query, args, err := sq.Select("value").
		From("preferences").
		Where(sq.Eq{"category": category, "key": key}).
		ToSql()
	if err != nil {
		return "", false, fmt.Errorf("build get query: %w", err)
	}

	err = s.db.QueryRow(query, args...).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get preference: %w", err)
	}
	return value, true, nil
}

func (s *Store) GetAll(category string) (map[string]string, error) {
	if err := validateCategory(category); err != nil {
		return nil, err
	}

	query, args, err := sq.Select("key", "value").
		From("preferences").
		Where(sq.Eq{"category": category}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build getall query: %w", err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("getall preferences: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan preference row: %w", err)
		}
		result[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate preferences: %w", err)
	}
	return result, nil
}

func (s *Store) ListCategories() ([]string, error) {
	query, args, err := sq.Select("DISTINCT category").
		From("preferences").
		OrderBy("category").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list categories query: %w", err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list preference categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan preference category: %w", err)
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate preference categories: %w", err)
	}
	return categories, nil
}

func (s *Store) ListKeys(category string) ([]string, error) {
	if err := validateCategory(category); err != nil {
		return nil, err
	}

	query, args, err := sq.Select("key").
		From("preferences").
		Where(sq.Eq{"category": category}).
		OrderBy("key").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build listkeys query: %w", err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list preference keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan preference key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate preference keys: %w", err)
	}
	return keys, nil
}

func (s *Store) Delete(category, key string) error {
	if err := validateCategoryKey(category, key); err != nil {
		return err
	}

	query, args, err := sq.Delete("preferences").
		Where(sq.Eq{"category": category, "key": key}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("delete preference: %w", err)
	}
	return nil
}

func (s *Store) DeleteCategory(category string) (int, error) {
	if err := validateCategory(category); err != nil {
		return 0, err
	}

	query, args, err := sq.Delete("preferences").
		Where(sq.Eq{"category": category}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete category query: %w", err)
	}

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete category: %w", err)
	}

	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete category rows affected: %w", err)
	}
	return int(removed), nil
}

func (s *Store) PurgeUnknown(category string, validKeys []string) (int, error) {
	if err := validateCategory(category); err != nil {
		return 0, err
	}
	if len(validKeys) == 0 {
		return 0, errEmptyValidKeys
	}

	query, args, err := sq.Delete("preferences").
		Where(sq.Eq{"category": category}).
		Where(sq.NotEq{"key": validKeys}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build purge query: %w", err)
	}

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("purge unknown preferences: %w", err)
	}

	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge rows affected: %w", err)
	}
	return int(removed), nil
}

func (s *Store) LoadAndPurgeUnknown(category string, validKeys []string) (map[string]string, error) {
	if _, err := s.PurgeUnknown(category, validKeys); err != nil {
		return nil, err
	}
	return s.GetAll(category)
}
