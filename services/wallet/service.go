package wallet

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/multiaccounts/accounts"
)

// NewService initializes service instance.
func NewService(db *Database, accountsFeed *event.Feed) *Service {
	feed := &event.Feed{}
	return &Service{
		db:   db,
		feed: feed,
		signals: &SignalsTransmitter{
			publisher: feed,
		},
		accountsFeed: accountsFeed,
	}
}

// Service is a wallet service.
type Service struct {
	feed                *event.Feed
	db                  *Database
	reactor             *Reactor
	signals             *SignalsTransmitter
	client              *ethclient.Client
	cryptoOnRampManager *CryptoOnRampManager
	started             bool

	group        *Group
	accountsFeed *event.Feed
}

// Start signals transmitter.
func (s *Service) Start(*p2p.Server) error {
	s.group = NewGroup(context.Background())
	return s.signals.Start()
}

// GetFeed returns signals feed.
func (s *Service) GetFeed() *event.Feed {
	return s.feed
}

// SetClient sets ethclient
func (s *Service) SetClient(client *ethclient.Client) {
	s.client = client
}

// MergeBlocksRanges merge old blocks ranges if possible
func (s *Service) MergeBlocksRanges(accounts []common.Address, chain uint64) error {
	for _, account := range accounts {
		err := s.db.mergeRanges(account, chain)
		if err != nil {
			return err
		}
	}

	return nil
}

// StartReactor separately because it requires known ethereum address, which will become available only after login.
func (s *Service) StartReactor(client *ethclient.Client, accounts []common.Address, chain *big.Int, watchNewBlocks bool) error {
	reactor := NewReactor(s.db, s.feed, client, chain, watchNewBlocks)
	err := reactor.Start(accounts)
	if err != nil {
		return err
	}
	s.reactor = reactor
	s.client = client
	s.group.Add(func(ctx context.Context) error {
		return WatchAccountsChanges(ctx, s.accountsFeed, accounts, reactor)
	})
	s.started = true
	return nil
}

// StopReactor stops reactor and closes database.
func (s *Service) StopReactor() error {
	if s.reactor == nil {
		return nil
	}
	s.reactor.Stop()
	if s.group != nil {
		s.group.Stop()
		s.group.Wait()
	}
	s.started = false
	return nil
}

// Stop reactor, signals transmitter and close db.
func (s *Service) Stop() error {
	log.Info("wallet will be stopped")
	err := s.StopReactor()
	s.signals.Stop()
	if s.group != nil {
		s.group.Stop()
		s.group.Wait()
		s.group = nil
	}
	log.Info("wallet stopped")
	return err
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "wallet",
			Version:   "0.1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

// WatchAccountsChanges subsribes to a feed and watches for changes in accounts list. If there are new or removed accounts
// reactor will be restarted.
func WatchAccountsChanges(ctx context.Context, feed *event.Feed, initial []common.Address, reactor *Reactor) error {
	accounts := make(chan []accounts.Account, 1) // it may block if the rate of updates will be significantly higher
	sub := feed.Subscribe(accounts)
	defer sub.Unsubscribe()
	listen := make(map[common.Address]struct{}, len(initial))
	for _, address := range initial {
		listen[address] = struct{}{}
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			if err != nil {
				log.Error("accounts watcher subscription failed", "error", err)
			}
		case n := <-accounts:
			log.Debug("wallet received updated list of accounts", "accounts", n)
			restart := false
			for _, acc := range n {
				_, exist := listen[common.Address(acc.Address)]
				if !exist {
					listen[common.Address(acc.Address)] = struct{}{}
					restart = true
				}
			}
			if !restart {
				continue
			}
			listenList := mapToList(listen)
			log.Debug("list of accounts was changed from a previous version. reactor will be restarted", "new", listenList)
			reactor.Stop()
			err := reactor.Start(listenList) // error is raised only if reactor is already running
			if err != nil {
				log.Error("failed to restart reactor with new accounts", "error", err)
			}
		}
	}
}

func mapToList(m map[common.Address]struct{}) []common.Address {
	rst := make([]common.Address, 0, len(m))
	for address := range m {
		rst = append(rst, address)
	}
	return rst
}

func (s *Service) IsStarted() bool {
	return s.started
}

func (s *Service) SetInitialBlocksRange(network uint64) error {
	accountsDB := accounts.NewDB(s.db.db)
	watchAddress, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}

	from := big.NewInt(0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	header, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	err = s.db.UpsertRange(common.Address(watchAddress), network, from, header.Number)
	if err != nil {
		return err
	}
	return nil
}
