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

	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/common"
)

type EthClientGetter interface {
	EthClient(chainID uint64) (chain.ClientInterface, error)
}

type LatestBlockData struct {
	blockNumber   uint64
	timestamp     time.Time
	blockDuration time.Duration
}

type BlockChainState struct {
	ethClientGetter    EthClientGetter
	mu                 sync.RWMutex
	latestBlockNumbers map[uint64]LatestBlockData
	sinceFn            func(time.Time) time.Duration
}

func NewBlockChainState(ethClientGetter EthClientGetter) *BlockChainState {
	return &BlockChainState{
		ethClientGetter:    ethClientGetter,
		mu:                 sync.RWMutex{},
		latestBlockNumbers: make(map[uint64]LatestBlockData),
		sinceFn:            time.Since,
	}
}

// Get the estimated latest block number for a chain
func (s *BlockChainState) GetEstimatedLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.estimateLatestBlockNumber(ctx, chainID)
}

// Get the estimated block time for a given block number and a chain
func (s *BlockChainState) GetEstimatedBlockTime(ctx context.Context, chainID uint64, blockNumber uint64) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.estimateBlockTime(ctx, chainID, blockNumber)
}

// Set the latest known block number for a chain
func (s *BlockChainState) SetLatestBlockNumber(chainID uint64, blockNumber uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.setLatestBlockNumber(chainID, blockNumber)
}

// Estimate the latest block number based on the last real value and the average block duration.
// Must be called with a locked mutex.
func (s *BlockChainState) estimateLatestBlockNumber(ctx context.Context, chainID uint64) (uint64, error) {
	blockData, err := s.getOrInitializeLatestBlockData(ctx, chainID)
	if err != nil {
		return 0, err
	}

	timeDiff := s.sinceFn(blockData.timestamp)
	blockDiff := uint64(math.Floor(float64(timeDiff) / float64(blockData.blockDuration)))
	return blockData.blockNumber + blockDiff, nil
}

// Estimate the timestamp for a given block number based on the last real value and the average block duration.
// Must be called with a locked mutex.
func (s *BlockChainState) estimateBlockTime(ctx context.Context, chainID uint64, blockNumber uint64) (time.Time, error) {
	blockData, err := s.getOrInitializeLatestBlockData(ctx, chainID)
	if err != nil {
		return time.Time{}, err
	}
	blockDiff := int64(blockNumber) - int64(blockData.blockNumber)
	timeDiff := time.Duration(blockDiff * int64(blockData.blockDuration))
	blockTime := blockData.timestamp.Add(timeDiff)
	return blockTime, nil
}

func (s *BlockChainState) getOrInitializeLatestBlockData(ctx context.Context, chainID uint64) (LatestBlockData, error) {
	blockData, ok := s.latestBlockNumbers[chainID]
	if !ok {
		err := s.initializeLatestBlockNumber(ctx, chainID)
		if err != nil {
			return LatestBlockData{}, err
		}
		blockData, ok = s.latestBlockNumbers[chainID]
		if !ok {
			return LatestBlockData{}, errors.New("Failed to initialize latest block number for chain ID " + strconv.Itoa(int(chainID)))
		}
	}
	return blockData, nil
}

// Initialize the latest block number for a chain
func (s *BlockChainState) initializeLatestBlockNumber(ctx context.Context, chainID uint64) error {
	ethClient, err := s.ethClientGetter.EthClient(chainID)
	if err != nil {
		return err
	}

	// Get multicall3 contract address
	multicallAddr, exists := multicall3.GetMulticall3Address(int64(chainID))
	if !exists {
		return errors.New("Multicall3 not supported on chain ID " + strconv.Itoa(int(chainID)))
	}

	// Create multicall3 contract instance for the caller interface
	multicallContract, err := multicall3.NewMulticall3(multicallAddr, ethClient)
	if err != nil {
		return err
	}

	blockNumber, err := multicallContract.GetBlockNumber(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return err
	}

	s.setLatestBlockNumber(chainID, blockNumber.Uint64())
	return nil
}

func (s *BlockChainState) setLatestBlockNumber(chainID uint64, blockNumber uint64) {
	blockDuration, found := common.AverageBlockDurationForChain[common.ChainID(chainID)]
	if !found {
		blockDuration = common.AverageBlockDurationForChain[common.ChainID(common.UnknownChainID)]
	}

	if data, ok := s.latestBlockNumbers[chainID]; ok {
		// Only update if the new block number is greater than the current one
		if data.blockNumber >= blockNumber {
			return
		}
	}

	s.latestBlockNumbers[chainID] = LatestBlockData{
		blockNumber:   blockNumber,
		timestamp:     time.Now(),
		blockDuration: blockDuration,
	}
}
