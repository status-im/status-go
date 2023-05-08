package transfer

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
)

type Controller struct {
	db                 *Database
	rpcClient          *rpc.Client
	blockDAO           *BlockDAO
	reactor            *Reactor
	accountFeed        *event.Feed
	TransferFeed       *event.Feed
	group              *async.Group
	transactionManager *TransactionManager
	fetchStrategyType  FetchStrategyType
}

func NewTransferController(db *sql.DB, rpcClient *rpc.Client, accountFeed *event.Feed, transferFeed *event.Feed,
	transactionManager *TransactionManager, fetchStrategyType FetchStrategyType) *Controller {

	blockDAO := &BlockDAO{db}
	return &Controller{
		db:                 NewDB(db),
		blockDAO:           blockDAO,
		rpcClient:          rpcClient,
		accountFeed:        accountFeed,
		TransferFeed:       transferFeed,
		transactionManager: transactionManager,
		fetchStrategyType:  fetchStrategyType,
	}
}

func (c *Controller) Start() {
	c.group = async.NewGroup(context.Background())
}

func (c *Controller) Stop() {
	if c.reactor != nil {
		c.reactor.stop()
	}

	if c.group != nil {
		c.group.Stop()
		c.group.Wait()
		c.group = nil
	}
}

func (c *Controller) SetInitialBlocksRange(chainIDs []uint64) error {
	chainClients, err := c.rpcClient.EthClients(chainIDs)
	if err != nil {
		return err
	}

	for chainID, chainClient := range chainClients {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		toHeader, err := chainClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return err
		}

		from := big.NewInt(0)

		err = c.blockDAO.setInitialBlocksRange(chainID, from, toHeader.Number)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) CheckRecentHistory(chainIDs []uint64, accounts []common.Address) error {
	if len(accounts) == 0 {
		log.Info("no accounts provided")
		return nil
	}

	if len(chainIDs) == 0 {
		log.Info("no chain provided")
		return nil
	}

	err := c.blockDAO.mergeBlocksRanges(chainIDs, accounts)
	if err != nil {
		return err
	}

	chainClients, err := c.rpcClient.EthClients(chainIDs)
	if err != nil {
		return err
	}

	if c.reactor != nil {
		err := c.reactor.restart(chainClients, accounts, c.fetchStrategyType)
		if err != nil {
			return err
		}
	} else {
		c.reactor = NewReactor(c.db, c.blockDAO, c.TransferFeed, c.transactionManager)

		err = c.reactor.start(chainClients, accounts, c.fetchStrategyType)
		if err != nil {
			return err
		}

		c.group.Add(func(ctx context.Context) error {
			return watchAccountsChanges(ctx, c.accountFeed, c.reactor, chainClients, accounts, c.fetchStrategyType)
		})
	}
	return nil
}

// watchAccountsChanges subscribes to a feed and watches for changes in accounts list. If there are new or removed accounts
// reactor will be restarted.
func watchAccountsChanges(ctx context.Context, accountFeed *event.Feed, reactor *Reactor,
	chainClients map[uint64]*chain.ClientWithFallback, initial []common.Address, fetchStrategyType FetchStrategyType) error {

	accounts := make(chan []*accounts.Account, 1) // it may block if the rate of updates will be significantly higher
	sub := accountFeed.Subscribe(accounts)
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

			err := reactor.restart(chainClients, listenList, fetchStrategyType)
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

func (c *Controller) LoadTransferByHash(ctx context.Context, rpcClient *rpc.Client, address common.Address, hash common.Hash) error {
	chainClient, err := rpcClient.EthClient(rpcClient.UpstreamChainID)
	if err != nil {
		return err
	}

	signer := types.NewLondonSigner(chainClient.ToBigInt())

	transfer, err := getTransferByHash(ctx, chainClient, signer, address, hash)
	if err != nil {
		return err
	}

	transfers := []Transfer{*transfer}

	err = c.db.InsertBlock(rpcClient.UpstreamChainID, address, transfer.BlockNumber, transfer.BlockHash)
	if err != nil {
		return err
	}

	blocks := []*big.Int{transfer.BlockNumber}
	err = c.db.SaveTransfersMarkBlocksLoaded(rpcClient.UpstreamChainID, address, transfers, blocks)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) GetTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int,
	limit int64, fetchMore bool) ([]View, error) {

	rst, err := c.reactor.getTransfersByAddress(ctx, chainID, address, toBlock, limit, fetchMore)
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	return castToTransferViews(rst), nil
}

func (c *Controller) GetCachedBalances(ctx context.Context, chainID uint64, addresses []common.Address) ([]BlockView, error) {
	result, error := c.blockDAO.getLastKnownBlocks(chainID, addresses)
	if error != nil {
		return nil, error
	}

	return blocksToViews(result), nil
}
