package multistandardbalance

//go:generate go tool mockgen -package=mock_multistandardbalance -source=fetcher.go -destination=mock/fetcher.go

import (
	"context"
	"errors"
	"strconv"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"

	"github.com/status-im/status-go/rpc/chain"
)

const DefaultBatchSize = 10000

type EthClientGetter interface {
	EthClient(chainID uint64) (chain.ClientInterface, error)
}

type Fetcher struct {
	ethClientGetter EthClientGetter
	batchSize       int
}

func NewFetcher(ethClientGetter EthClientGetter, batchSize int) *Fetcher {
	return &Fetcher{
		ethClientGetter: ethClientGetter,
		batchSize:       batchSize,
	}
}

func (f *Fetcher) FetchBalances(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
	ethClient, err := f.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	// Get multicall3 contract address
	multicallAddr, exists := multicall3.GetMulticall3Address(int64(chainID))
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
