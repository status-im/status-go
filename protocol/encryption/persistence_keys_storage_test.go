package encryption

import (
	"path/filepath"
	"testing"

	dr "github.com/status-im/doubleratchet"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/sqlite"
)

var (
	pubKey1 = dr.Key{0xe3, 0xbe, 0xb9, 0x4e, 0x70, 0x17, 0x37, 0xc, 0x1, 0x8f, 0xa9, 0x7e, 0xef, 0x4, 0xfb, 0x23, 0xac, 0xea, 0x28, 0xf7, 0xa9, 0x56, 0xcc, 0x1d, 0x46, 0xf3, 0xb5, 0x1d, 0x7d, 0x7d, 0x5e, 0x2c}
	pubKey2 = dr.Key{0xec, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20, 0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a, 0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb, 0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40}
	mk1     = dr.Key{0x00, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20, 0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a, 0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb, 0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40}
	mk2     = dr.Key{0x01, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20, 0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a, 0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb, 0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40}
	mk3     = dr.Key{0x02, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20, 0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a, 0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb, 0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40}
	mk4     = dr.Key{0x03, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20, 0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a, 0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb, 0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40}
	mk5     = dr.Key{0x04, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20, 0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a, 0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb, 0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40}
)

func TestSQLLitePersistenceKeysStorageTestSuite(t *testing.T) {
	suite.Run(t, new(SQLLitePersistenceKeysStorageTestSuite))
}

type SQLLitePersistenceKeysStorageTestSuite struct {
	suite.Suite
	service dr.KeysStorage
}

func (s *SQLLitePersistenceKeysStorageTestSuite) SetupTest() {
	dir := s.T().TempDir()
	key := "blahblahblah"

	db, err := sqlite.Open(filepath.Join(dir, "db.sql"), key, sqlite.ReducedKDFIterationsNumber)
	s.Require().NoError(err)

	p := newSQLitePersistence(db)
	s.service = p.KeysStorage()
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLiteGetMissing() {
	// Act.
	_, ok, err := s.service.Get(pubKey1, 0)

	// Assert.
	s.NoError(err)
	s.False(ok, "It returns false")
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLite_Put() {
	// Act and assert.
	err := s.service.Put([]byte("session-id"), pubKey1, 0, mk1, 1)
	s.NoError(err)
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLite_DeleteOldMks() {
	// Insert keys out-of-order
	err := s.service.Put([]byte("session-id"), pubKey1, 0, mk1, 1)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey1, 1, mk2, 2)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey1, 2, mk3, 20)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey1, 3, mk4, 21)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey1, 4, mk5, 22)
	s.NoError(err)

	err = s.service.DeleteOldMks([]byte("session-id"), 20)
	s.NoError(err)

	_, ok, err := s.service.Get(pubKey1, 0)
	s.NoError(err)
	s.False(ok)

	_, ok, err = s.service.Get(pubKey1, 1)
	s.NoError(err)
	s.False(ok)

	_, ok, err = s.service.Get(pubKey1, 2)
	s.NoError(err)
	s.False(ok)

	_, ok, err = s.service.Get(pubKey1, 3)
	s.NoError(err)
	s.True(ok)

	_, ok, err = s.service.Get(pubKey1, 4)
	s.NoError(err)
	s.True(ok)
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLite_TruncateMks() {
	// Insert keys out-of-order
	err := s.service.Put([]byte("session-id"), pubKey2, 2, mk5, 5)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey2, 0, mk3, 3)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey1, 1, mk2, 2)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey2, 1, mk4, 4)
	s.NoError(err)
	err = s.service.Put([]byte("session-id"), pubKey1, 0, mk1, 1)
	s.NoError(err)

	err = s.service.TruncateMks([]byte("session-id"), 2)
	s.NoError(err)

	_, ok, err := s.service.Get(pubKey1, 0)
	s.NoError(err)
	s.False(ok)

	_, ok, err = s.service.Get(pubKey1, 1)
	s.NoError(err)
	s.False(ok)

	_, ok, err = s.service.Get(pubKey2, 0)
	s.NoError(err)
	s.False(ok)

	_, ok, err = s.service.Get(pubKey2, 1)
	s.NoError(err)
	s.True(ok)

	_, ok, err = s.service.Get(pubKey2, 2)
	s.NoError(err)
	s.True(ok)
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLite_Count() {

	// Act.
	cnt, err := s.service.Count(pubKey1)

	// Assert.
	s.NoError(err)
	s.EqualValues(0, cnt, "It returns 0 when no keys are in the database")
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLite_Delete() {
	// Arrange.

	// Act and assert.
	err := s.service.DeleteMk(pubKey1, 0)
	s.NoError(err)
}

func (s *SQLLitePersistenceKeysStorageTestSuite) TestKeysStorageSqlLite_Flow() {

	// Act.
	err := s.service.Put([]byte("session-id"), pubKey1, 0, mk1, 1)
	s.NoError(err)

	k, ok, err := s.service.Get(pubKey1, 0)

	// Assert.
	s.NoError(err)
	s.True(ok, "It returns true")
	s.Equal(mk1, k, "It returns the message key")

	// Act.
	_, ok, err = s.service.Get(pubKey2, 0)

	// Assert.
	s.NoError(err)
	s.False(ok, "It returns false when querying non existing public key")

	// Act.
	_, ok, err = s.service.Get(pubKey1, 1)

	// Assert.
	s.NoError(err)
	s.False(ok, "It returns false when querying the wrong msg number")

	// Act.
	cnt, err := s.service.Count(pubKey1)

	// Assert.
	s.NoError(err)
	s.EqualValues(1, cnt)

	// Act and assert.
	err = s.service.DeleteMk(pubKey1, 1)
	s.NoError(err)

	// Act and assert.
	err = s.service.DeleteMk(pubKey2, 0)
	s.NoError(err)

	// Act.
	err = s.service.DeleteMk(pubKey1, 0)
	s.NoError(err)

	cnt, err = s.service.Count(pubKey1)

	// Assert.
	s.NoError(err)
	s.EqualValues(0, cnt)
}
