package wallet

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/migrations"
	"github.com/status-im/status-go/sqlite"
)

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path string) (*Database, error) {
	db, err := sqlite.OpenDB(path)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

// NullBytes type for downloading potentially null bytes,
// FIXME(dshulyak) replace with JSONBlob for all cases.
type NullBytes struct {
	Bytes []byte
	Valid bool
}

// Scan implements interface.
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

// Value implement interface.
func (b *NullBytes) Value() (driver.Value, error) {
	if !b.Valid {
		return nil, nil
	}
	return b.Bytes, nil
}

// SQLBigInt type for storing uin256 in the databse.
// FIXME(dshulyak) SQL bit int is max 64 bits. Maybe store as bytes in big endian and hope
// that lexographical sorting will work.
type SQLBigInt big.Int

// Scan implements interface.
func (i *SQLBigInt) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("not an integer")
	}
	(*big.Int)(i).SetInt64(val)
	return nil
}

// Value implements interface.
func (i *SQLBigInt) Value() (driver.Value, error) {
	if !(*big.Int)(i).IsInt64() {
		return nil, errors.New("not at int64")
	}
	return (*big.Int)(i).Int64(), nil
}

// JSONBlob type for marshaling/unmarshaling inner type to json.
type JSONBlob struct {
	Interface interface{}
}

// Scan implements interface.
func (blob *JSONBlob) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("not a byte slice")
	}
	if len(bytes) == 0 {
		return nil
	}
	err := json.Unmarshal(bytes, blob.Interface)
	return err
}

// Value implements interface.
func (blob *JSONBlob) Value() (driver.Value, error) {
	return json.Marshal(blob.Interface)
}

// Database sql wrapper for operations with wallet objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

// ProcessTranfers atomically adds/removes blocks and adds new tranfers.
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
		_, err = blocks.Exec(header.Hash(), (*SQLBigInt)(header.Number), &JSONBlob{header})
		if err != nil {
			return err
		}
	}

	for _, t := range transfers {
		_, err = insert.Exec(t.Transaction.Hash(), t.Header.Hash(), &JSONBlob{t.Transaction}, &JSONBlob{t.Receipt}, t.Type)
		if err != nil {
			return err
		}
	}
	return err
}

// GetTransfers load transfers transfer betweeen two blocks.
func (db *Database) GetTransfers(start, end *big.Int) (rst []Transfer, err error) {
	query := "SELECT type, blocks.header, tx, receipt FROM transfers JOIN blocks ON blk_hash = blocks.hash WHERE blocks.number >= ?"
	var (
		rows *sql.Rows
	)
	if end != nil {
		query += " AND blocks.number <= ?"
		rows, err = db.db.Query(query, (*SQLBigInt)(start), (*SQLBigInt)(end))
	} else {
		rows, err = db.db.Query(query, (*SQLBigInt)(start))
	}
	if err != nil {
		return
	}
	for rows.Next() {
		transfer := Transfer{
			Header:      &types.Header{},
			Transaction: &types.Transaction{},
			Receipt:     &types.Receipt{},
		}
		err = rows.Scan(
			&transfer.Type, &JSONBlob{transfer.Header},
			&JSONBlob{transfer.Transaction}, &JSONBlob{transfer.Receipt})
		if err != nil {
			return nil, err
		}
		rst = append(rst, transfer)
	}
	return
}

// SaveHeader stores single header.
func (db *Database) SaveHeader(header *types.Header) error {
	headerJSON, err := header.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = db.db.Exec("INSERT INTO blocks(number, hash, header) VALUES (?, ?, ?)", (*SQLBigInt)(header.Number), header.Hash(), headerJSON)
	return err
}

// SaveHeaders atomically stores list of headers.
func (db *Database) SaveHeaders(headers []*types.Header) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
		buf    []byte
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	insert, err = tx.Prepare("INSERT INTO blocks(number,hash,header) VALUES (?,?,?)")
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()

	for _, h := range headers {
		buf, err = h.MarshalJSON()
		if err != nil {
			return
		}
		_, err = insert.Exec((*SQLBigInt)(h.Number), h.Hash(), buf)
		if err != nil {
			return
		}
	}
	return
}

// LastHeader selects last header by block number.
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

// HeaderExists checks if header with hash exists in db.
func (db *Database) HeaderExists(hash common.Hash) (bool, error) {
	var val sql.NullBool
	err := db.db.QueryRow("SELECT EXISTS (SELECT hash FROM blocks WHERE hash = ?)", hash).Scan(&val)
	if err != nil {
		return false, err
	}
	return val.Bool, nil
}

// GetHeaderByNumber selects header using block number.
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
