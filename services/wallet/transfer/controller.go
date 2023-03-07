package transfer

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

type Controller struct {
	db                 *Database
	rpcClient          *rpc.Client
	block              *Block
	reactor            *Reactor
	accountFeed        *event.Feed
	TransferFeed       *event.Feed
	group              *async.Group
	balanceCache       *balanceCache
	transactionManager *TransactionManager
}

func NewTransferController(db *sql.DB, rpcClient *rpc.Client, accountFeed *event.Feed, transferFeed *event.Feed, transactionManager *TransactionManager) *Controller {
	block := &Block{db}
	return &Controller{
		db:                 NewDB(db),
		block:              block,
		rpcClient:          rpcClient,
		accountFeed:        accountFeed,
		TransferFeed:       transferFeed,
		transactionManager: transactionManager,
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

	for _, chainClient := range chainClients {
		err := c.block.setInitialBlocksRange(chainClient)
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

	err := c.block.mergeBlocksRanges(chainIDs, accounts)
	if err != nil {
		return err
	}

	chainClients, err := c.rpcClient.EthClients(chainIDs)
	if err != nil {
		return err
	}

	if c.reactor != nil {
		err := c.reactor.restart(chainClients, accounts)
		if err != nil {
			return err
		}
	}

	c.reactor = &Reactor{
		db:                 c.db,
		feed:               c.TransferFeed,
		block:              c.block,
		transactionManager: c.transactionManager,
	}
	err = c.reactor.start(chainClients, accounts)
	if err != nil {
		return err
	}

	c.group.Add(func(ctx context.Context) error {
		return watchAccountsChanges(ctx, c.accountFeed, c.reactor, chainClients, accounts)
	})
	return nil
}

// watchAccountsChanges subscribes to a feed and watches for changes in accounts list. If there are new or removed accounts
// reactor will be restarted.
func watchAccountsChanges(ctx context.Context, accountFeed *event.Feed, reactor *Reactor, chainClients []*chain.ClientWithFallback, initial []common.Address) error {
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
			err := reactor.restart(chainClients, listenList)
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
	err = c.db.SaveTransfers(rpcClient.UpstreamChainID, address, transfers, blocks)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) GetTransfersByAddress(ctx context.Context, chainID uint64, address common.Address, toBlock *big.Int, limit int64, fetchMore bool) ([]View, error) {
	log.Debug("[WalletAPI:: GetTransfersByAddress] get transfers for an address", "address", address)

	rst, err := c.db.GetTransfersByAddress(chainID, address, toBlock, limit)
	if err != nil {
		log.Error("[WalletAPI:: GetTransfersByAddress] can't fetch transfers", "err", err)
		return nil, err
	}

	transfersCount := int64(len(rst))
	chainClient, err := c.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	if fetchMore && limit > transfersCount {
		block, err := c.block.GetFirstKnownBlock(chainID, address)
		if err != nil {
			return nil, err
		}

		// if zero block was already checked there is nothing to find more
		if block == nil || big.NewInt(0).Cmp(block) == 0 {
			return castToTransferViews(rst), nil
		}

		from, err := findFirstRange(ctx, address, block, chainClient)
		if err != nil {
			if nonArchivalNodeError(err) {
				c.TransferFeed.Send(walletevent.Event{
					Type: EventNonArchivalNodeDetected,
				})
				from = big.NewInt(0).Sub(block, big.NewInt(100))
			} else {
				log.Error("first range error", "error", err)
				return nil, err
			}
		}
		fromByAddress := map[common.Address]*LastKnownBlock{address: {
			Number: from,
		}}
		toByAddress := map[common.Address]*big.Int{address: block}

		if c.balanceCache == nil {
			c.balanceCache = newBalanceCache()
		}
		blocksCommand := &findAndCheckBlockRangeCommand{
			accounts:      []common.Address{address},
			db:            c.db,
			chainClient:   chainClient,
			balanceCache:  c.balanceCache,
			feed:          c.TransferFeed,
			fromByAddress: fromByAddress,
			toByAddress:   toByAddress,
		}

		if err = blocksCommand.Command()(ctx); err != nil {
			return nil, err
		}

		blocks, err := c.block.GetBlocksByAddress(chainID, address, numberOfBlocksCheckedPerIteration)
		if err != nil {
			return nil, err
		}

		log.Info("checking blocks again", "blocks", len(blocks))
		if len(blocks) > 0 {
			txCommand := &loadTransfersCommand{
				accounts:           []common.Address{address},
				db:                 c.db,
				block:              c.block,
				chainClient:        chainClient,
				transactionManager: c.transactionManager,
			}

			err = txCommand.Command()(ctx)
			if err != nil {
				return nil, err
			}

			rst, err = c.db.GetTransfersByAddress(chainID, address, toBlock, limit)
			if err != nil {
				return nil, err
			}
		}
	}

	return castToTransferViews(rst), nil
}

func (c *Controller) GetCachedBalances(ctx context.Context, chainID uint64, addresses []common.Address) ([]LastKnownBlockView, error) {
	result, error := c.block.getLastKnownBalances(chainID, addresses)
	if error != nil {
		return nil, error
	}

	return blocksToViews(result), nil
}
