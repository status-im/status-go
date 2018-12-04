package mailserver

import (
	"testing"

	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type panicDB struct{}

func (db *panicDB) Close() error {
	panic("panicDB panic on Close")
}

func (db *panicDB) Write(b *leveldb.Batch, opts *opt.WriteOptions) error {
	panic("panicDB panic on Write")
}

func (db *panicDB) Put(k []byte, v []byte, opts *opt.WriteOptions) error {
	panic("panicDB panic on Put")
}

func (db *panicDB) Get(k []byte, opts *opt.ReadOptions) ([]byte, error) {
	panic("panicDB panic on Get")
}

func (db *panicDB) NewIterator(r *util.Range, opts *opt.ReadOptions) iterator.Iterator {
	panic("panicDB panic on NewIterator")
}

func TestMailServerDBPanicSuite(t *testing.T) {
	suite.Run(t, new(MailServerDBPanicSuite))
}

type MailServerDBPanicSuite struct {
	suite.Suite
	server *WMailServer
}

func (s *MailServerDBPanicSuite) SetupTest() {
	s.server = &WMailServer{}
	s.server.db = &panicDB{}
}

func (s *MailServerDBPanicSuite) TestArchive() {
	defer s.testPanicRecover("Archive")
	s.server.Archive(&whisper.Envelope{})
}

func (s *MailServerDBPanicSuite) TestDeliverMail() {
	defer s.testPanicRecover("DeliverMail")
	s.server.DeliverMail(&whisper.Peer{}, &whisper.Envelope{})
}

func (s *MailServerDBPanicSuite) testPanicRecover(method string) {
	if r := recover(); r != nil {
		s.Failf("error recovering panic", "expected recover to return nil, got: %+v", r)
	}
}
