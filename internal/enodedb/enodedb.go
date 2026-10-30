// Package enodedb sizes go-ethereum's in-memory node database.
//
// enode.OpenDB("") opens goleveldb with default options, which preallocates a
// 4MB memtable that stays live for the node's lifetime. The node database only
// holds the local ENR sequence and discovery bookkeeping, so a small write
// buffer is plenty. go-ethereum offers no option for this, and enode's
// database is the only goleveldb instance status-go opens, so the package
// default is lowered instead.
package enodedb

import "github.com/syndtr/goleveldb/leveldb/opt"

const writeBuffer = 64 * opt.KiB

func init() {
	opt.DefaultWriteBuffer = writeBuffer
}
