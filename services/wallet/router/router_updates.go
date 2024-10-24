package router

import (
	"context"
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc/chain"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

var (
	newBlockCheckIntervalMainnet      = 3 * time.Second
	newBlockCheckIntervalOptimism     = 1 * time.Second
	newBlockCheckIntervalArbitrum     = 200 * time.Millisecond
	newBlockCheckIntervalAnvilMainnet = 2 * time.Second

	feeRecalculationTimeout      = 5 * time.Minute
	feeRecalculationAnvilTimeout = 5 * time.Second
)

type fetchingLastBlock struct {
	client    chain.ClientInterface
	lastBlock uint64
	closeCh   chan struct{}
}

func (r *Router) subscribeForUdates(chainID uint64) error {
	if _, ok := r.clientsForUpdatesPerChains.Load(chainID); ok {
		return nil
	}

	ethClient, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error("Failed to get eth client", zap.Error(err))
		return err
	}

	flb := fetchingLastBlock{
		client:    ethClient,
		lastBlock: 0,
		closeCh:   make(chan struct{}),
	}
	r.clientsForUpdatesPerChains.Store(chainID, flb)

	timeout := feeRecalculationTimeout
	if chainID == walletCommon.AnvilMainnet {
		timeout = feeRecalculationAnvilTimeout
	}
	r.startTimeoutForUpdates(flb.closeCh, timeout)

	var ticker *time.Ticker
	switch chainID {
	case walletCommon.EthereumMainnet,
		walletCommon.EthereumSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalMainnet)
	case walletCommon.OptimismMainnet,
		walletCommon.OptimismSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalOptimism)
	case walletCommon.ArbitrumMainnet,
		walletCommon.ArbitrumSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalArbitrum)
	case walletCommon.AnvilMainnet:
		ticker = time.NewTicker(newBlockCheckIntervalAnvilMainnet)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())

	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case <-ticker.C:
				var blockNumber uint64
				blockNumber, err := ethClient.BlockNumber(ctx)
				if err != nil {
					logutils.ZapLogger().Error("Failed to get block number", zap.Error(err))
					continue
				}

				val, ok := r.clientsForUpdatesPerChains.Load(chainID)
				if !ok {
					logutils.ZapLogger().Error("Failed to get fetchingLastBlock", zap.Uint64("chain", chainID))
					continue
				}

				flbLoaded, ok := val.(fetchingLastBlock)
				if !ok {
					logutils.ZapLogger().Error("Failed to get fetchingLastBlock", zap.Uint64("chain", chainID))
					continue
				}

				if blockNumber > flbLoaded.lastBlock {
					flbLoaded.lastBlock = blockNumber
					r.clientsForUpdatesPerChains.Store(chainID, flbLoaded)

					fees, err := r.feesManager.SuggestedFees(ctx, chainID)
					if err != nil {
						logutils.ZapLogger().Error("Failed to get suggested fees", zap.Error(err))
						continue
					}

					r.lastInputParamsMutex.Lock()
					uuid := r.lastInputParams.Uuid
					r.lastInputParamsMutex.Unlock()

					r.activeRoutesMutex.Lock()
					if r.activeRoutes != nil && r.activeRoutes.Best != nil && len(r.activeRoutes.Best) > 0 {
						for _, path := range r.activeRoutes.Best {
							err = r.cacluateFees(ctx, path, fees, false, 0)
							if err != nil {
								logutils.ZapLogger().Error("Failed to calculate fees", zap.Error(err))
								continue
							}
						}

						_, err = r.checkBalancesForTheBestRoute(ctx, r.activeRoutes.Best)

						sendRouterResult(uuid, r.activeRoutes, err)
					}
					r.activeRoutesMutex.Unlock()
				}
			case <-flb.closeCh:
				ticker.Stop()
				cancelCtx()
				return
			}
		}
	}()
	return nil
}

func (r *Router) startTimeoutForUpdates(closeCh chan struct{}, timeout time.Duration) {
	dedlineTicker := time.NewTicker(timeout)
	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case <-dedlineTicker.C:
				r.unsubscribeFeesUpdateAccrossAllChains()
				return
			case <-closeCh:
				dedlineTicker.Stop()
				return
			}
		}
	}()
}

func (r *Router) unsubscribeFeesUpdateAccrossAllChains() {
	r.clientsForUpdatesPerChains.Range(func(key, value interface{}) bool {
		flb, ok := value.(fetchingLastBlock)
		if !ok {
			logutils.ZapLogger().Error("Failed to get fetchingLastBlock", zap.Any("chain", key))
			return false
		}

		close(flb.closeCh)
		r.clientsForUpdatesPerChains.Delete(key)
		return true
	})
}
