package wallet

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/common"
)

const (
	fetchLatestBlockNumbersInterval = 10 * time.Minute
)

type fetchLatestBlockNumberCommand struct {
	state      *BlockChainState
	rpcClient  *rpc.Client
	accountsDB *accounts.Database
}

func (c *fetchLatestBlockNumberCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: fetchLatestBlockNumbersInterval,
		Runable:  c.Run,
	}.Run
}

func (c *fetchLatestBlockNumberCommand) Run(parent context.Context) (err error) {
	log.Debug("start fetchLatestBlockNumberCommand")

	networks, err := c.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil
	}
	areTestNetworksEnabled, err := c.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return
	}
	ctx := context.Background()
	for _, network := range networks {
		if network.IsTest != areTestNetworksEnabled {
			continue
		}
		_, _ = c.state.fetchLatestBlockNumber(ctx, network.ChainID)
	}
	return nil
}

type LatestBlockData struct {
	blockNumber   uint64
	timestamp     time.Time
	blockDuration time.Duration
}

type BlockChainState struct {
	rpcClient          *rpc.Client
	accountsDB         *accounts.Database
	blkMu              sync.RWMutex
	latestBlockNumbers map[uint64]LatestBlockData
	group              *async.Group
	cancelFn           context.CancelFunc
	sinceFn            func(time.Time) time.Duration
}

func NewBlockChainState(rpcClient *rpc.Client, accountsDb *accounts.Database) *BlockChainState {
	return &BlockChainState{
		rpcClient:          rpcClient,
		accountsDB:         accountsDb,
		blkMu:              sync.RWMutex{},
		latestBlockNumbers: make(map[uint64]LatestBlockData),
		sinceFn:            time.Since,
	}
}

func (s *BlockChainState) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel
	s.group = async.NewGroup(ctx)

	command := &fetchLatestBlockNumberCommand{
		state:      s,
		accountsDB: s.accountsDB,
		rpcClient:  s.rpcClient,
	}
	s.group.Add(command.Command())
}

func (s *BlockChainState) Stop() {
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
	if s.group != nil {
		s.group.Stop()
		s.group.Wait()
		s.group = nil
	}
}

func (s *BlockChainState) GetEstimatedLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	blockNumber, ok := s.estimateLatestBlockNumber(chainID)
	if ok {
		return blockNumber, nil
	}
	return s.fetchLatestBlockNumber(ctx, chainID)
}

func (s *BlockChainState) fetchLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	client, err := s.rpcClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	blockDuration, found := common.AverageBlockDurationForChain[common.ChainID(chainID)]
	if !found {
		blockDuration = common.AverageBlockDurationForChain[common.ChainID(common.UnknownChainID)]
	}
	s.setLatestBlockDataForChain(chainID, LatestBlockData{
		blockNumber:   blockNumber,
		timestamp:     time.Now(),
		blockDuration: blockDuration,
	})
	return blockNumber, nil
}

func (s *BlockChainState) setLatestBlockDataForChain(chainID uint64, latestBlockData LatestBlockData) {
	s.blkMu.Lock()
	defer s.blkMu.Unlock()
	s.latestBlockNumbers[chainID] = latestBlockData
}

func (s *BlockChainState) estimateLatestBlockNumber(chainID uint64) (uint64, bool) {
	s.blkMu.RLock()
	defer s.blkMu.RUnlock()
	blockData, ok := s.latestBlockNumbers[chainID]
	if !ok {
		return 0, false
	}
	timeDiff := s.sinceFn(blockData.timestamp)
	return blockData.blockNumber + uint64((timeDiff / blockData.blockDuration)), true
}
