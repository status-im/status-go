package enodedb

import (
	"net"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"

	wakuenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"

	"github.com/status-im/status-go/internal/testutils/goheap"
)

type EnodeDBSuite struct {
	suite.Suite
}

func TestEnodeDBSuite(t *testing.T) {
	suite.Run(t, new(EnodeDBSuite))
}

func (s *EnodeDBSuite) TestLocalNodeDoesNotRetainDefaultWriteBuffer() {
	defer func(rate int) { runtime.MemProfileRate = rate }(runtime.MemProfileRate)
	runtime.MemProfileRate = 1

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	localNode, err := wakuenr.NewLocalnode(key)
	s.Require().NoError(err)
	defer localNode.Database().Close()

	s.Require().Less(goheap.LiveBytesAllocatedBy("leveldb/memdb.New"), int64(1<<20))
	runtime.KeepAlive(localNode)
}

func (s *EnodeDBSuite) TestLocalNodeDatabaseStoresNodes() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	localNode, err := wakuenr.NewLocalnode(key)
	s.Require().NoError(err)
	defer localNode.Database().Close()

	db := localNode.Database()
	for i := 0; i < 500; i++ {
		peerKey, err := crypto.GenerateKey()
		s.Require().NoError(err)
		var record enr.Record
		record.Set(enr.IP(net.IPv4(10, 0, byte(i>>8), byte(i))))
		record.Set(enr.UDP(30303))
		s.Require().NoError(enode.SignV4(&record, peerKey))
		peer, err := enode.New(enode.ValidSchemes, &record)
		s.Require().NoError(err)
		s.Require().NoError(db.UpdateNode(peer))

		stored := db.Node(peer.ID())
		s.Require().NotNil(stored)
		s.Require().Equal(peer.Seq(), stored.Seq())
	}
}
