package multistandardbalance

//go:generate go tool mockgen -package=mock_multistandardbalance -source=fetcher.go -destination=mock/fetcher.go

import (
	"context"
	"errors"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"

	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
)

const DefaultBatchSize = 10000

type EthClientGetter interface {
	EthClient(chainID uint64) (ethclient.EthClientInterface, error)
}

type Fetcher struct {
	ethClientGetter    EthClientGetter
	batchSize          int
	multicallOverrides map[uint64]common.Address
}

func NewFetcher(ethClientGetter EthClientGetter, batchSize int, multicallOverrides map[uint64]common.Address) *Fetcher {
	return &Fetcher{
		ethClientGetter:    ethClientGetter,
		batchSize:          batchSize,
		multicallOverrides: multicallOverrides,
	}
}

func (f *Fetcher) getMulticall3Address(chainID uint64) (common.Address, bool) {
	multicallAddr, exists := f.multicallOverrides[chainID]
	if exists {
		return multicallAddr, true
	}
	return multicall3.GetMulticall3Address(int64(chainID))
}

func (f *Fetcher) FetchBalances(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
	ethClient, err := f.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	// Get multicall3 contract address
	multicallAddr, exists := f.getMulticall3Address(chainID)
	if !exists {
		return nil, errors.New("Multicall3 not supported on chain ID " + strconv.Itoa(int(chainID)))
	}

	// Create multicall3 contract instance for the caller interface
	multicallContract, err := multicall3.NewMulticall3(multicallAddr, ethClient)
	if err != nil {
		return nil, err
	}

	resultsCh := multistandardfetcher.FetchBalances(ctx, multicallAddr, multicallContract, config, f.batchSize)

	return resultsCh, nil
}
