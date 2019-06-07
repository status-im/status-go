package wallet

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/t/devtests/testchain"
	"github.com/stretchr/testify/suite"
)

func TestNewBlocksSuite(t *testing.T) {
	suite.Run(t, new(NewBlocksSuite))
}

type NewBlocksSuite struct {
	suite.Suite
	backend *testchain.Backend
	cmd     *newBlocksTransfersCommand
	address common.Address
	db      *Database
	dbStop  func()
	feed    *event.Feed
}

func (s *NewBlocksSuite) SetupTest() {
	var err error
	db, stop := setupTestDB(s.Suite.T())
	s.db = db
	s.dbStop = stop
	s.backend, err = testchain.NewBackend()
	s.Require().NoError(err)
	account, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.address = crypto.PubkeyToAddress(account.PublicKey)
	s.feed = &event.Feed{}
	s.cmd = &newBlocksTransfersCommand{
		db:       s.db,
		accounts: []common.Address{s.address},
		erc20:    NewERC20TransfersDownloader(s.backend.Client, []common.Address{s.address}),
		eth: &ETHTransferDownloader{
			client:   s.backend.Client,
			signer:   s.backend.Signer,
			accounts: []common.Address{s.address},
		},
		feed:        s.feed,
		client:      s.backend.Client,
		safetyDepth: big.NewInt(15),
	}
}

func (s *NewBlocksSuite) TearDownTest() {
	s.dbStop()
	s.Require().NoError(s.backend.Stop())
}

func (s *NewBlocksSuite) TestOneBlock() {
	ctx := context.Background()
	s.Require().EqualError(s.cmd.Run(ctx), "not found")
	tx := types.NewTransaction(0, s.address, big.NewInt(1e17), 21000, big.NewInt(1), nil)
	tx, err := types.SignTx(tx, s.backend.Signer, s.backend.Faucet)
	s.Require().NoError(err)
	blocks := s.backend.GenerateBlocks(1, 0, func(n int, gen *core.BlockGen) {
		gen.AddTx(tx)
	})
	n, err := s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().Equal(1, n)
	s.Require().NoError(err)

	events := make(chan Event, 1)
	sub := s.feed.Subscribe(events)
	defer sub.Unsubscribe()

	s.Require().NoError(s.cmd.Run(ctx))

	select {
	case ev := <-events:
		s.Require().Equal(ev.Type, EventNewBlock)
		s.Require().Equal(ev.BlockNumber, big.NewInt(1))
	default:
		s.Require().FailNow("event wasn't emitted")
	}
	transfers, err := s.db.GetTransfers(big.NewInt(0), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
	s.Require().Equal(tx.Hash(), transfers[0].ID)
}

func (s *NewBlocksSuite) genTx(nonce int) *types.Transaction {
	tx := types.NewTransaction(uint64(nonce), s.address, big.NewInt(1e10), 21000, big.NewInt(1), nil)
	tx, err := types.SignTx(tx, s.backend.Signer, s.backend.Faucet)
	s.Require().NoError(err)
	return tx
}

func (s *NewBlocksSuite) runCmdUntilError(ctx context.Context) (err error) {
	for err == nil {
		err = s.cmd.Run(ctx)
	}
	return err
}

func (s *NewBlocksSuite) TestReorg() {
	blocks := s.backend.GenerateBlocks(20, 0, nil)
	n, err := s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().Equal(20, n)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Require().EqualError(s.runCmdUntilError(ctx), "not found")

	blocks = s.backend.GenerateBlocks(3, 20, func(n int, gen *core.BlockGen) {
		gen.AddTx(s.genTx(n))
	})
	n, err = s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().Equal(3, n)
	s.Require().NoError(err)

	// `not found` returned when we query head+1 block
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Require().EqualError(s.runCmdUntilError(ctx), "not found")

	transfers, err := s.db.GetTransfers(big.NewInt(0), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 3)

	blocks = s.backend.GenerateBlocks(10, 15, func(n int, gen *core.BlockGen) {
		gen.AddTx(s.genTx(n))
	})
	n, err = s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().Equal(10, n)
	s.Require().NoError(err)

	// it will be less but even if something wrong we can't get more
	events := make(chan Event, 10)
	sub := s.feed.Subscribe(events)
	defer sub.Unsubscribe()

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Require().EqualError(s.runCmdUntilError(ctx), "not found")

	close(events)
	expected := []Event{{Type: EventReorg, BlockNumber: big.NewInt(16)}, {Type: EventNewBlock, BlockNumber: big.NewInt(25)}}
	i := 0
	for ev := range events {
		s.Require().Equal(expected[i].Type, ev.Type)
		s.Require().Equal(expected[i].BlockNumber, ev.BlockNumber)
		i++
	}

	transfers, err = s.db.GetTransfers(big.NewInt(0), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 10)
}

func (s *NewBlocksSuite) downloadHistorical() {
	blocks := s.backend.GenerateBlocks(40, 0, func(n int, gen *core.BlockGen) {
		if n == 36 {
			gen.AddTx(s.genTx(0))
		} else if n == 39 {
			gen.AddTx(s.genTx(1))
		}
	})
	n, err := s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().Equal(40, n)
	s.Require().NoError(err)

	eth := &ethHistoricalCommand{
		db: s.db,
		eth: &ETHTransferDownloader{
			client:   s.backend.Client,
			signer:   s.backend.Signer,
			accounts: []common.Address{s.address},
		},
		feed:        s.feed,
		address:     s.address,
		client:      s.backend.Client,
		safetyDepth: big.NewInt(0),
	}
	s.Require().NoError(eth.Run(context.Background()), "eth historical command failed to sync transfers")
	transfers, err := s.db.GetTransfers(big.NewInt(0), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 2)
}

func (s *NewBlocksSuite) reorgHistorical() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Require().EqualError(s.runCmdUntilError(ctx), "not found")

	blocks := s.backend.GenerateBlocks(10, 35, nil)
	n, err := s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().Equal(10, n)
	s.Require().NoError(err)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Require().EqualError(s.runCmdUntilError(ctx), "not found")

}

func (s *NewBlocksSuite) TestSafetyBufferFailure() {
	s.downloadHistorical()

	s.cmd.safetyDepth = big.NewInt(0)
	s.reorgHistorical()

	transfers, err := s.db.GetTransfers(big.NewInt(0), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func (s *NewBlocksSuite) TestSafetyBufferSuccess() {
	s.downloadHistorical()

	s.cmd.safetyDepth = big.NewInt(10)
	s.reorgHistorical()

	transfers, err := s.db.GetTransfers(big.NewInt(0), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 0)
}
