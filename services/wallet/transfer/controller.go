package transfer

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

type Controller struct {
	db           *Database
	rpcClient    *rpc.Client
	signals      *SignalsTransmitter
	block        *Block
	reactor      *Reactor
	accountFeed  *event.Feed
	TransferFeed *event.Feed
	group        *async.Group
	balanceCache *balanceCache
}

func NewTransferController(db *sql.DB, rpcClient *rpc.Client, accountFeed *event.Feed) *Controller {
	transferFeed := &event.Feed{}
	signals := &SignalsTransmitter{
		publisher: transferFeed,
	}
	block := &Block{db}
	return &Controller{
		db:           NewDB(db),
		block:        block,
		rpcClient:    rpcClient,
		signals:      signals,
		accountFeed:  accountFeed,
		TransferFeed: transferFeed,
	}
}

func (c *Controller) Start() error {
	c.group = async.NewGroup(context.Background())
	return c.signals.Start()
}

func (c *Controller) Stop() {
	c.signals.Stop()
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
	chainClients, err := chain.NewClients(c.rpcClient, chainIDs)
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

	chainClients, err := chain.NewClients(c.rpcClient, chainIDs)
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
		db:    c.db,
		feed:  c.TransferFeed,
		block: c.block,
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

// watchAccountsChanges subsribes to a feed and watches for changes in accounts list. If there are new or removed accounts
// reactor will be restarted.
func watchAccountsChanges(ctx context.Context, accountFeed *event.Feed, reactor *Reactor, chainClients []*chain.Client, initial []common.Address) error {
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
	chainClient, err := chain.NewClient(rpcClient, rpcClient.UpstreamChainID)
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
	err = c.db.SaveTranfers(rpcClient.UpstreamChainID, address, transfers, blocks)
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
	chainClient, err := chain.NewClient(c.rpcClient, chainID)
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
				accounts:    []common.Address{address},
				db:          c.db,
				block:       c.block,
				chainClient: chainClient,
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

type BalanceState struct {
	Value     *hexutil.Big `json:"value"`
	Timestamp uint64       `json:"time"`
}

type BalanceHistoryTimeInterval int

const (
	BalanceHistory7Hours BalanceHistoryTimeInterval = iota + 1
	BalanceHistory1Month
	BalanceHistory6Months
	BalanceHistory1Year
	BalanceHistoryAllTime
)

var balanceHistoryTimeIntervalToHoursPerStep = map[BalanceHistoryTimeInterval]int64{
	BalanceHistory7Hours:  2,
	BalanceHistory1Month:  12,
	BalanceHistory6Months: (24 * 7) / 2,
	BalanceHistory1Year:   24 * 7,
}

var balanceHistoryTimeIntervalToSampleNo = map[BalanceHistoryTimeInterval]int64{
	BalanceHistory7Hours:  84,
	BalanceHistory1Month:  60,
	BalanceHistory6Months: 52,
	BalanceHistory1Year:   52,
	BalanceHistoryAllTime: 50,
}

// GetBalanceHistory expect a time precision of +/- average block time (~12s)
// implementation relies that a block has constant time length to save block header requests
func (c *Controller) GetBalanceHistory(ctx context.Context, chainID uint64, address common.Address, timeInterval BalanceHistoryTimeInterval) ([]BalanceState, error) {
	chainClient, err := chain.NewClient(c.rpcClient, chainID)
	if err != nil {
		return nil, err
	}

	if c.balanceCache == nil {
		c.balanceCache = newBalanceCache()
	}

	if c.balanceCache.history == nil {
		c.balanceCache.history = new(balanceHistoryCache)
	}

	currentTimestamp := time.Now().Unix()
	lastBlockNo := big.NewInt(0)
	var lastBlockTimestamp int64
	if (currentTimestamp - c.balanceCache.history.lastBlockTimestamp) >= (12 * 60 * 60) {
		lastBlock, err := chainClient.BlockByNumber(ctx, nil)
		if err != nil {
			return nil, err
		}
		lastBlockNo.Set(lastBlock.Number())
		lastBlockTimestamp = int64(lastBlock.Time())
		c.balanceCache.history.lastBlockNo = big.NewInt(0).Set(lastBlockNo)
		c.balanceCache.history.lastBlockTimestamp = lastBlockTimestamp
	} else {
		lastBlockNo.Set(c.balanceCache.history.lastBlockNo)
		lastBlockTimestamp = c.balanceCache.history.lastBlockTimestamp
	}

	initialBlock, err := chainClient.BlockByNumber(ctx, big.NewInt(1))
	if err != nil {
		return nil, err
	}
	initialBlockNo := big.NewInt(0).Set(initialBlock.Number())
	initialBlockTimestamp := int64(initialBlock.Time())

	allTimeBlockCount := big.NewInt(0).Sub(lastBlockNo, initialBlockNo)
	allTimeInterval := lastBlockTimestamp - initialBlockTimestamp

	// Expected to be around 12
	blockDuration := float64(allTimeInterval) / float64(allTimeBlockCount.Int64())

	lastBlockTime := time.Unix(lastBlockTimestamp, 0)
	// Snap to the beginning of the day or half day which is the closest to the last block
	hour := 0
	if lastBlockTime.Hour() >= 12 {
		hour = 12
	}
	lastTime := time.Date(lastBlockTime.Year(), lastBlockTime.Month(), lastBlockTime.Day(), hour, 0, 0, 0, lastBlockTime.Location())
	endBlockTimestamp := lastTime.Unix()
	blockGaps := big.NewInt(int64(float64(lastBlockTimestamp-endBlockTimestamp) / blockDuration))
	endBlockNo := big.NewInt(0).Sub(lastBlockNo, blockGaps)

	totalBlockCount, startTimestamp := int64(0), int64(0)
	if timeInterval == BalanceHistoryAllTime {
		startTimestamp = initialBlockTimestamp
		totalBlockCount = endBlockNo.Int64()
	} else {
		secondsToNow := balanceHistoryTimeIntervalToHoursPerStep[timeInterval] * 3600 * (balanceHistoryTimeIntervalToSampleNo[timeInterval])
		startTimestamp = endBlockTimestamp - secondsToNow
		totalBlockCount = int64(float64(secondsToNow) / blockDuration)
	}
	blocksInStep := totalBlockCount / (balanceHistoryTimeIntervalToSampleNo[timeInterval])
	stepDuration := int64(float64(blocksInStep) * blockDuration)

	points := make([]BalanceState, 0)

	nextBlockNumber := big.NewInt(0).Set(endBlockNo)
	nextTimestamp := endBlockTimestamp
	for nextTimestamp >= startTimestamp && nextBlockNumber.Cmp(initialBlockNo) >= 0 && nextBlockNumber.Cmp(big.NewInt(0)) > 0 {
		newBlockNo := big.NewInt(0).Set(nextBlockNumber)
		currentBalance, err := c.balanceCache.BalanceAt(ctx, chainClient, address, newBlockNo)
		if err != nil {
			return nil, err
		}

		var currentBalanceState BalanceState
		currentBalanceState.Value = (*hexutil.Big)(currentBalance)
		currentBalanceState.Timestamp = uint64(nextTimestamp)
		points = append([]BalanceState{currentBalanceState}, points...)

		// decrease block number and timestamp
		nextTimestamp -= stepDuration
		nextBlockNumber.Sub(nextBlockNumber, big.NewInt(blocksInStep))
	}
	return points, nil
}
