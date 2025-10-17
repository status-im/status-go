package transferdetector

//go:generate go tool mockgen -package=mock_transferdetector -source=fetcher.go -destination=mock/fetcher.go

import (
	"context"

	"github.com/status-im/go-wallet-sdk/pkg/eventfilter"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/rpc/chain/ethclient"
)

type EthClientGetter interface {
	EthClient(chainID uint64) (ethclient.EthClientInterface, error)
}

type Fetcher struct {
	ethClientGetter EthClientGetter
}

func NewFetcher(ethClientGetter EthClientGetter) *Fetcher {
	return &Fetcher{
		ethClientGetter: ethClientGetter,
	}
}

func (f *Fetcher) FilterTransfers(ctx context.Context, chainID uint64, config eventfilter.TransferQueryConfig) ([]eventlog.Event, error) {
	ethClient, err := f.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return eventfilter.FilterTransfers(ctx, ethClient, config)
}
