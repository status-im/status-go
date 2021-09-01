package wallet

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

// NewService initializes service instance.
func NewService(db *sql.DB, chainID uint64, accountsFeed *event.Feed) *Service {
	feed := &event.Feed{}
	signals := &SignalsTransmitter{
		publisher: feed,
	}
	cryptoOnRampManager := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceStatic,
	})
	tokenManager := &TokenManager{db: db}
	transactionManager := &TransactionManager{db: db}
	networkManager := &NetworkManager{db: db, chainClients: make(map[uint64]*chainClient)}
	err := networkManager.init()
	if err != nil {
		log.Error("Network manager failed to initialize", "error", err)
	}

	return &Service{
		feed:                feed,
		db:                  NewDB(db),
		networkManager:      networkManager,
		tokenManager:        tokenManager,
		transactionManager:  transactionManager,
		accountsFeed:        accountsFeed,
		opensea:             newOpenseaClient(),
		signals:             signals,
		cryptoOnRampManager: cryptoOnRampManager,
		legacyChainID:       chainID,
	}
}

// Service is a wallet service.
type Service struct {
	feed                *event.Feed
	networkManager      *NetworkManager
	tokenManager        *TokenManager
	transactionManager  *TransactionManager
	db                  *Database
	reactor             *Reactor
	signals             *SignalsTransmitter
	cryptoOnRampManager *CryptoOnRampManager
	opensea             *OpenseaClient
	group               *Group
	accountsFeed        *event.Feed
	legacyChainID       uint64
	started             bool
}

// Start signals transmitter.
func (s *Service) Start() error {
	s.group = NewGroup(context.Background())
	return s.signals.Start()
}

// GetFeed returns signals feed.
func (s *Service) GetFeed() *event.Feed {
	return s.feed
}

// StartReactor separately because it requires known ethereum address, which will become available only after login.
func (s *Service) StartReactor(accounts []common.Address, chainID uint64) error {
	chainClient, err := s.networkManager.getChainClient(chainID)
	if err != nil {
		return err
	}

	reactor := NewReactor(s.db, s.feed, chainClient, chainID)
	err = reactor.Start(accounts)
	if err != nil {
		return err
	}

	s.reactor = reactor
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

func (s *Service) SetInitialBlocksRange(chainID uint64) error {
	db := s.db
	accountsDB := accounts.NewDB(db.client)
	watchAddress, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}

	from := big.NewInt(0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	chainClient, err := s.networkManager.getChainClient(chainID)
	if err != nil {
		return err
	}

	header, err := chainClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	err = db.InsertRange(chainID, common.Address(watchAddress), from, header.Number, big.NewInt(0), 0)
	if err != nil {
		return err
	}
	return nil
}

// MergeBlocksRanges merge old blocks ranges if possible
func (s *Service) MergeBlocksRanges(accounts []common.Address, chainID uint64) error {
	for _, account := range accounts {
		err := s.db.mergeRanges(chainID, account)
		if err != nil {
			return err
		}
	}

	return nil
}
