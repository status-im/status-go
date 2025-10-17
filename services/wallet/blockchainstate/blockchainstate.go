package blockchainstate

//go:generate go tool mockgen -package=mock_blockchainstate -source=blockchainstate.go -destination=mock/blockchainstate.go

import (
	"context"
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"

	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/services/wallet/common"
)

type EthClientGetter interface {
	EthClient(chainID uint64) (ethclient.EthClientInterface, error)
}

type LatestBlockData struct {
	blockNumber uint64
	timestamp   time.Time
}

type BlockChainState struct {
	ethClientGetter EthClientGetter
	latestBlockData map[uint64]LatestBlockData // map[chainID]*LatestBlockData
	sinceFn         func(time.Time) time.Duration
	blockDurationFn func(chainID uint64) time.Duration
	mu              sync.RWMutex
}

func NewBlockChainState(ethClientGetter EthClientGetter) *BlockChainState {
	return &BlockChainState{
		ethClientGetter: ethClientGetter,
		latestBlockData: make(map[uint64]LatestBlockData),
		sinceFn:         time.Since,
		blockDurationFn: func(chainID uint64) time.Duration {
			blockDuration, found := common.AverageBlockDurationForChain[common.ChainID(chainID)]
			if !found {
				blockDuration = common.AverageBlockDurationForChain[common.ChainID(common.UnknownChainID)]
			}
			return blockDuration
		},
	}
}

// Estimate the latest block number based on the last real value and the average block duration.
func (s *BlockChainState) GetEstimatedLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	blockData, err := s.getOrInitializeLatestBlockData(ctx, chainID)
	if err != nil {
		return 0, err
	}

	timeDiff := s.sinceFn(blockData.timestamp)
	blockDuration := s.blockDurationFn(chainID)
	blockDiff := uint64(math.Floor(float64(timeDiff) / float64(blockDuration)))
	return blockData.blockNumber + blockDiff, nil
}

// Estimate the timestamp for a given block number based on the last real value and the average block duration.
func (s *BlockChainState) GetEstimatedBlockTime(ctx context.Context, chainID uint64, blockNumber uint64) (time.Time, error) {
	blockData, err := s.getOrInitializeLatestBlockData(ctx, chainID)
	if err != nil {
		return time.Time{}, err
	}
	blockDiff := int64(blockNumber) - int64(blockData.blockNumber)
	blockDuration := s.blockDurationFn(chainID)
	timeDiff := time.Duration(blockDiff * int64(blockDuration))
	blockTime := blockData.timestamp.Add(timeDiff)
	return blockTime, nil
}

// Set the latest known block number for a chain
func (s *BlockChainState) SetLatestBlockNumber(chainID uint64, blockNumber uint64) {
	s.setLatestBlockNumber(chainID, blockNumber)
}

func (s *BlockChainState) getOrInitializeLatestBlockData(ctx context.Context, chainID uint64) (LatestBlockData, error) {
	// Try read lock first for fast path
	s.mu.RLock()
	blockData, exists := s.latestBlockData[chainID]
	s.mu.RUnlock()
	if exists {
		return blockData, nil
	}

	// Not found, need to initialize
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check after acquiring write lock
	blockData, exists = s.latestBlockData[chainID]
	if exists {
		return blockData, nil
	}

	blockNumber, err := s.getLatestBlockNumber(ctx, chainID)
	if err != nil {
		return LatestBlockData{}, err
	}
	s.latestBlockData[chainID] = buildLatestBlockData(blockNumber)
	return s.latestBlockData[chainID], nil
}

// Get the latest block number for a chain
func (s *BlockChainState) getLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	ethClient, err := s.ethClientGetter.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	// Get multicall3 contract address
	multicallAddr, exists := multicall3.GetMulticall3Address(int64(chainID))
	if !exists {
		return 0, errors.New("Multicall3 not supported on chain ID " + strconv.Itoa(int(chainID)))
	}

	// Create multicall3 contract instance for the caller interface
	multicallContract, err := multicall3.NewMulticall3(multicallAddr, ethClient)
	if err != nil {
		return 0, err
	}

	blockNumber, err := multicallContract.GetBlockNumber(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return 0, err
	}

	return blockNumber.Uint64(), nil
}

func (s *BlockChainState) setLatestBlockNumber(chainID uint64, blockNumber uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldData, exists := s.latestBlockData[chainID]
	if !exists {
		// No old value, initialized with newData
		s.latestBlockData[chainID] = buildLatestBlockData(blockNumber)
		return
	}
	if oldData.blockNumber >= blockNumber {
		// Only update if the new block number is greater than the current one
		return
	}
	// Update the existing value
	s.latestBlockData[chainID] = buildLatestBlockData(blockNumber)
}

func buildLatestBlockData(blockNumber uint64) LatestBlockData {
	return LatestBlockData{
		blockNumber: blockNumber,
		timestamp:   time.Now(),
	}
}
