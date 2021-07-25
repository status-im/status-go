package sqlds

import (
	"context"

	ds "github.com/ipfs/go-datastore"
)

type op struct {
	delete bool
	value  []byte
}

type batch struct {
	ds  *Datastore
	ops map[ds.Key]op
}

// Batch creates a set of deferred updates to the database.
// Since SQL does not support a true batch of updates,
// operations are buffered and then executed sequentially
// over a single connection when Commit is called.
func (d *Datastore) Batch() (ds.Batch, error) {
	return &batch{
		ds:  d,
		ops: make(map[ds.Key]op),
	}, nil
}

func (bt *batch) Put(key ds.Key, val []byte) error {
	bt.ops[key] = op{value: val}
	return nil
}

func (bt *batch) Delete(key ds.Key) error {
	bt.ops[key] = op{delete: true}
	return nil
}

func (bt *batch) Commit() error {
	return bt.CommitContext(context.Background())
}

func (bt *batch) CommitContext(ctx context.Context) error {
	conn, err := bt.ds.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	for k, op := range bt.ops {
		if op.delete {
			_, err = conn.ExecContext(ctx, bt.ds.queries.Delete(), k.String())
		} else {
			_, err = conn.ExecContext(ctx, bt.ds.queries.Put(), k.String(), op.value)
		}
		if err != nil {
			break
		}
	}

	return err
}

var _ ds.Batching = (*Datastore)(nil)
