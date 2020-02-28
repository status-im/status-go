package wallet

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/t/devtests/testchain"
	"github.com/status-im/status-go/t/utils"

	"github.com/status-im/status-go/eth-node/types"
)

func TestReactorChanges(t *testing.T) {
	utils.Init()
	suite.Run(t, new(ReactorChangesSuite))
}

type ReactorChangesSuite struct {
	suite.Suite
	backend *testchain.Backend
	reactor *Reactor
	db      *Database
	dbStop  func()
	feed    *event.Feed

	first, second common.Address
}

func (s *ReactorChangesSuite) txToAddress(nonce uint64, address common.Address) *gethtypes.Transaction {
	tx := gethtypes.NewTransaction(nonce, address, big.NewInt(1e17), 21000, big.NewInt(1), nil)
	tx, err := gethtypes.SignTx(tx, s.backend.Signer, s.backend.Faucet)
	s.Require().NoError(err)
	return tx
}

func (s *ReactorChangesSuite) SetupTest() {
	var err error
	db, stop := setupTestDB(s.Suite.T())
	s.db = db
	s.dbStop = stop
	s.backend, err = testchain.NewBackend()
	s.Require().NoError(err)
	s.feed = &event.Feed{}
	s.reactor = NewReactor(s.db, &event.Feed{}, s.backend.Client, big.NewInt(1337))
	account, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.first = crypto.PubkeyToAddress(account.PublicKey)
	account, err = crypto.GenerateKey()
	s.Require().NoError(err)
	s.second = crypto.PubkeyToAddress(account.PublicKey)
	nonce := uint64(0)
	blocks := s.backend.GenerateBlocks(1, 0, func(n int, gen *core.BlockGen) {
		gen.AddTx(s.txToAddress(nonce, s.first))
		nonce++
		gen.AddTx(s.txToAddress(nonce, s.second))
		nonce++
	})
	_, err = s.backend.Ethereum.BlockChain().InsertChain(blocks)
	s.Require().NoError(err)
}

func (s *ReactorChangesSuite) TestWatchNewAccounts() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	group := NewGroup(ctx)
	group.Add(func(ctx context.Context) error {
		return WatchAccountsChanges(ctx, s.feed, []common.Address{s.first}, s.reactor)
	})
	s.Require().NoError(s.reactor.Start([]common.Address{s.first}))
	s.Require().NoError(utils.Eventually(func() error {
		transfers, err := s.db.GetTransfersInRange(s.first, big.NewInt(0), nil)
		if err != nil {
			return err
		}
		if len(transfers) != 1 {
			return fmt.Errorf("expect to get 1 transfer for first address %x, got %d", s.first, len(transfers))
		}
		transfers, err = s.db.GetTransfersInRange(s.second, big.NewInt(0), nil)
		if err != nil {
			return err
		}
		if len(transfers) != 0 {
			return fmt.Errorf("expect not to get any transfer for second address %x", s.second)
		}
		return nil
	}, 5*time.Second, 500*time.Millisecond))
	s.feed.Send([]accounts.Account{{Address: types.Address(s.first)}, {Address: types.Address(s.second)}})
	s.Require().NoError(utils.Eventually(func() error {
		transfers, err := s.db.GetTransfersInRange(s.second, big.NewInt(0), nil)
		if err != nil {
			return err
		}
		if len(transfers) == 0 {
			return fmt.Errorf("expect 1 transfer for second address %x, got %d", s.second, len(transfers))
		}
		return nil
	}, 5*time.Second, 500*time.Millisecond))
}

func TestServiceStartStop(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	backend, err := testchain.NewBackend()
	require.NoError(t, err)

	s := NewService(db, &event.Feed{})
	require.NoError(t, s.Start(nil))

	account, err := crypto.GenerateKey()
	require.NoError(t, err)

	err = s.StartReactor(backend.Client, []common.Address{crypto.PubkeyToAddress(account.PublicKey)}, big.NewInt(1337))
	require.NoError(t, err)

	require.NoError(t, s.Stop())
	require.NoError(t, s.Start(nil))

	err = s.StartReactor(backend.Client, []common.Address{crypto.PubkeyToAddress(account.PublicKey)}, big.NewInt(1337))
	require.NoError(t, err)
	require.NoError(t, s.Stop())
}
