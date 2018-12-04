package mailservers

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/db"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// NewPeerRecord returns instance of the peer record.
func NewPeerRecord(node *enode.Node) PeerRecord {
	return PeerRecord{node: node}
}

// PeerRecord is set data associated with each peer that is stored on disk.
type PeerRecord struct {
	node *enode.Node

	// last time it was used.
	LastUsed time.Time
}

// Encode encodes PeerRecords to bytes.
func (r PeerRecord) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// Node returns a pointer to a enode.Node object unmarshalled from key.
func (r PeerRecord) Node() *enode.Node {
	return r.node
}

// NewCache returns pointer to a Cache instance.
func NewCache(db *leveldb.DB) *Cache {
	return &Cache{db: db}
}

// Cache is wrapper for operations on disk with leveldb.
type Cache struct {
	db *leveldb.DB
}

// Replace delets old and adds new records in the persistent cache.
func (c *Cache) Replace(nodes []*enode.Node) error {
	batch := new(leveldb.Batch)
	iter := c.db.NewIterator(util.BytesPrefix([]byte{byte(db.MailserversCache)}), nil)
	defer iter.Release()
	newNodes := nodesToMap(nodes)
	for iter.Next() {
		record, err := unmarshalKeyValue(iter.Key()[1:], iter.Value())
		if err != nil {
			return err
		}
		if _, exist := newNodes[record.Node().ID()]; exist {
			delete(newNodes, record.Node().ID())
		} else {
			batch.Delete(iter.Key())
		}
	}
	for _, n := range newNodes {
		enodeKey, err := n.MarshalText()
		if err != nil {
			return err
		}
		batch.Put(db.Key(db.MailserversCache, enodeKey), nil)
	}
	return c.db.Write(batch, nil)
}

// LoadAll loads all records from persistent database.
func (c *Cache) LoadAll() (rst []PeerRecord, err error) {
	iter := c.db.NewIterator(util.BytesPrefix([]byte{byte(db.MailserversCache)}), nil)
	for iter.Next() {
		record, err := unmarshalKeyValue(iter.Key()[1:], iter.Value())
		if err != nil {
			return nil, err
		}
		rst = append(rst, record)
	}
	return rst, nil
}

// UpdateRecord updates single record.
func (c *Cache) UpdateRecord(record PeerRecord) error {
	enodeKey, err := record.Node().MarshalText()
	if err != nil {
		return err
	}
	value, err := record.Encode()
	if err != nil {
		return err
	}
	return c.db.Put(db.Key(db.MailserversCache, enodeKey), value, nil)
}

func unmarshalKeyValue(key, value []byte) (record PeerRecord, err error) {
	enodeKey := key
	node := new(enode.Node)
	err = node.UnmarshalText(enodeKey)
	if err != nil {
		return record, err
	}
	record = PeerRecord{node: node}
	if len(value) != 0 {
		err = json.Unmarshal(value, &record)
	}
	return record, err
}

func nodesToMap(nodes []*enode.Node) map[enode.ID]*enode.Node {
	rst := map[enode.ID]*enode.Node{}
	for _, n := range nodes {
		rst[n.ID()] = n
	}
	return rst
}
