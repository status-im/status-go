package sqlds

import (
	"database/sql"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

// Queries generates SQL queries for datastore operations.
type Queries interface {
	Delete() string
	Exists() string
	Get() string
	Put() string
	Query() string
	Prefix() string
	Limit() string
	Offset() string
	GetSize() string
}

// Datastore is a SQL backed datastore.
type Datastore struct {
	db      *sql.DB
	queries Queries
}

// NewDatastore returns a new SQL datastore.
func NewDatastore(db *sql.DB, queries Queries) *Datastore {
	return &Datastore{db: db, queries: queries}
}

// Close closes the underying SQL database.
func (d *Datastore) Close() error {
	return d.db.Close()
}

// Delete removes a row from the SQL database by the given key.
func (d *Datastore) Delete(key ds.Key) error {
	_, err := d.db.Exec(d.queries.Delete(), key.String())
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a value from the SQL database by the given key.
func (d *Datastore) Get(key ds.Key) (value []byte, err error) {
	row := d.db.QueryRow(d.queries.Get(), key.String())
	var out []byte

	switch err := row.Scan(&out); err {
	case sql.ErrNoRows:
		return nil, ds.ErrNotFound
	case nil:
		return out, nil
	default:
		return nil, err
	}
}

// Has determines if a value for the given key exists in the SQL database.
func (d *Datastore) Has(key ds.Key) (exists bool, err error) {
	row := d.db.QueryRow(d.queries.Exists(), key.String())

	switch err := row.Scan(&exists); err {
	case sql.ErrNoRows:
		return exists, nil
	case nil:
		return exists, nil
	default:
		return exists, err
	}
}

// Put "upserts" a row into the SQL database.
func (d *Datastore) Put(key ds.Key, value []byte) error {
	_, err := d.db.Exec(d.queries.Put(), key.String(), value)
	if err != nil {
		return err
	}

	return nil
}

// Query returns multiple rows from the SQL database based on the passed query parameters.
func (d *Datastore) Query(q dsq.Query) (dsq.Results, error) {
	raw, err := d.rawQuery(q)
	if err != nil {
		return nil, err
	}

	for _, f := range q.Filters {
		raw = dsq.NaiveFilter(raw, f)
	}

	raw = dsq.NaiveOrder(raw, q.Orders...)

	// if we have filters or orders, offset and limit won't have been applied in the query
	if len(q.Filters) > 0 || len(q.Orders) > 0 {
		if q.Offset != 0 {
			raw = dsq.NaiveOffset(raw, q.Offset)
		}
		if q.Limit != 0 {
			raw = dsq.NaiveLimit(raw, q.Limit)
		}
	}

	return raw, nil
}

func (d *Datastore) rawQuery(q dsq.Query) (dsq.Results, error) {
	var rows *sql.Rows
	var err error

	rows, err = queryWithParams(d, q)
	if err != nil {
		return nil, err
	}

	it := dsq.Iterator{
		Next: func() (dsq.Result, bool) {
			if !rows.Next() {
				return dsq.Result{}, false
			}

			var key string
			var out []byte

			err := rows.Scan(&key, &out)
			if err != nil {
				return dsq.Result{Error: err}, false
			}

			entry := dsq.Entry{Key: key}

			if !q.KeysOnly {
				entry.Value = out
			}
			if q.ReturnsSizes {
				entry.Size = len(out)
			}

			return dsq.Result{Entry: entry}, true
		},
		Close: func() error {
			return rows.Close()
		},
	}

	return dsq.ResultsFromIterator(q, it), nil
}

// Sync is noop for SQL databases.
func (d *Datastore) Sync(key ds.Key) error {
	return nil
}

// GetSize determines the size in bytes of the value for a given key.
func (d *Datastore) GetSize(key ds.Key) (int, error) {
	row := d.db.QueryRow(d.queries.GetSize(), key.String())
	var size int

	switch err := row.Scan(&size); err {
	case sql.ErrNoRows:
		return -1, ds.ErrNotFound
	case nil:
		return size, nil
	default:
		return 0, err
	}
}

// queryWithParams applies prefix, limit, and offset params in pg query
func queryWithParams(d *Datastore, q dsq.Query) (*sql.Rows, error) {
	var qNew = d.queries.Query()

	if q.Prefix != "" {
		// normalize
		prefix := ds.NewKey(q.Prefix).String()
		if prefix != "/" {
			qNew += fmt.Sprintf(d.queries.Prefix(), prefix+"/")
		}
	}

	// only apply limit and offset if we do not have to naive filter/order the results
	if len(q.Filters) == 0 && len(q.Orders) == 0 {
		if q.Limit != 0 {
			qNew += fmt.Sprintf(d.queries.Limit(), q.Limit)
		}
		if q.Offset != 0 {
			qNew += fmt.Sprintf(d.queries.Offset(), q.Offset)
		}
	}

	return d.db.Query(qNew)

}

var _ ds.Datastore = (*Datastore)(nil)
