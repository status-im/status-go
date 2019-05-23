package wallet

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/migrate"
	"github.com/status-im/migrate/database/sqlcipher"
	bindata "github.com/status-im/migrate/source/go_bindata"
	"github.com/status-im/status-go/services/wallet/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Migrate applies migrations.
func Migrate(db *sql.DB) error {
	resources := bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func InitializeDB(path string) (*Database, error) {
	db, err := sqlite.OpenDB(path)
	if err != nil {
		return nil, err
	}
	err = Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

type NullBytes struct {
	Bytes []byte
	Valid bool
}

func (b *NullBytes) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	buf, ok := value.([]byte)
	if !ok {
		return errors.New("value not bytes")
	}
	b.Bytes = make([]byte, len(buf))
	copy(b.Bytes, buf)
	b.Valid = true
	return nil
}

func (b *NullBytes) Value() (driver.Value, error) {
	if !b.Valid {
		return nil, nil
	}
	return b.Bytes, nil
}

type SQLBigInt big.Int

func (i *SQLBigInt) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("not an integer")
	}
	(*big.Int)(i).SetInt64(val)
	return nil
}

func (i *SQLBigInt) Value() (driver.Value, error) {
	if !(*big.Int)(i).IsInt64() {
		return nil, errors.New("not at int64")
	}
	return (*big.Int)(i).Int64(), nil
}

type Database struct {
	db *sql.DB
}

func (db Database) Close() error {
	return db.db.Close()
}

func (db Database) ProcessTranfers(transfers []Transfer, added, removed []*types.Header) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
		blocks *sql.Stmt
		delete *sql.Stmt
	)
	tx, err = db.db.Begin()
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

	insert, err = tx.Prepare("INSERT INTO transfers(hash, blk_hash, tx, receipt, type) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	delete, err = tx.Prepare("DELETE FROM blocks WHERE hash = ?")
	if err != nil {
		return err
	}
	blocks, err = tx.Prepare("INSERT INTO blocks(hash,number,header) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	for _, header := range removed {
		_, err = delete.Exec(header.Hash())
		if err != nil {
			return err
		}
	}
	for _, header := range added {
		headerJSON, _ := header.MarshalJSON()
		_, err = blocks.Exec(header.Hash(), (*SQLBigInt)(header.Number), headerJSON)
		if err != nil {
			return err
		}
	}

	for _, t := range transfers {
		txJSON, _ := t.Transaction.MarshalJSON()
		receiptJSON, _ := t.Receipt.MarshalJSON()
		_, err = insert.Exec(t.Transaction.Hash(), t.Header.Hash(), txJSON, receiptJSON, t.Type)
		if err != nil {
			return err
		}
	}
	return err
}

func (db *Database) SaveHeader(header *types.Header) error {
	headerJSON, err := header.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = db.db.Exec("INSERT INTO blocks(number, hash, header) VALUES (?, ?, ?)", (*SQLBigInt)(header.Number), header.Hash(), headerJSON)
	return err
}

func (db *Database) LastHeader() (*types.Header, error) {
	var buf NullBytes
	err := db.db.QueryRow("SELECT header FROM blocks WHERE number = (SELECT MAX(number) FROM blocks)").Scan(&buf)
	if err != nil {
		return nil, err
	}
	if !buf.Valid {
		return nil, errors.New("not found")
	}
	header := &types.Header{}
	err = header.UnmarshalJSON(buf.Bytes)
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (db *Database) HeaderExists(hash common.Hash) (bool, error) {
	var val sql.NullBool
	err := db.db.QueryRow("SELECT EXISTS (SELECT hash FROM blocks WHERE hash = ?)", hash).Scan(&val)
	if err != nil {
		return false, err
	}
	return val.Bool, nil
}

func (db *Database) GetHeaderByNumber(number *big.Int) (*types.Header, error) {
	var buf NullBytes
	err := db.db.QueryRow("SELECT header FROM blocks WHERE number = ?", (*SQLBigInt)(number)).Scan(&buf)
	if err != nil {
		return nil, err
	}
	if !buf.Valid {
		return nil, errors.New("not found")
	}
	header := &types.Header{}
	err = header.UnmarshalJSON(buf.Bytes)
	if err != nil {
		return nil, err
	}
	return header, nil
}
