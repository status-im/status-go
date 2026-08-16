package router

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

var (
	newBlockCheckIntervalMainnet      = 3 * time.Second
	newBlockCheckIntervalOptimism     = 1 * time.Second
	newBlockCheckIntervalArbitrum     = 200 * time.Millisecond
	newBlockCheckIntervalBase         = 1 * time.Second
	newBlockCheckIntervalLinea        = 1 * time.Second
	newBlockCheckIntervalUnichain     = 1 * time.Second
	newBlockCheckIntervalKatana       = 1 * time.Second
	newBlockCheckIntervalInk          = 1 * time.Second
	newBlockCheckIntervalAbstract     = 200 * time.Millisecond
	newBlockCheckIntervalZkSync       = 1 * time.Second
	newBlockCheckIntervalSoneium      = 2 * time.Second
	newBlockCheckIntervalScroll       = 1 * time.Second
	newBlockCheckIntervalBlast        = 2 * time.Second
	newBlockCheckIntervalRobinhood    = 1 * time.Second
	newBlockCheckIntervalBSC          = 3 * time.Second
	newBlockCheckIntervalAnvilMainnet = 2 * time.Second

	feeRecalculationTimeout      = 5 * time.Minute
	feeRecalculationAnvilTimeout = 5 * time.Second
)

type fetchingLastBlock struct {
	client    ethclient.EthClientInterface
	lastBlock uint64
	closeCh   chan struct{}
}

func (r *Router) subscribeForUdates(chainID uint64, address common.Address) error {
	if _, ok := r.clientsForUpdatesPerChains.Load(chainID); ok {
		r.logger.Debug("subscribeForUdates: chain already subscribed", zap.Uint64("chainId", chainID))
		return nil
	}
	r.logger.Info("subscribeForUdates: subscribing for fee updates",
		zap.Uint64("chainId", chainID),
		zap.Stringer("address", address))

	ethClient, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		r.logger.Error("subscribeForUdates: failed to get eth client",
			zap.Uint64("chainId", chainID),
			zap.Error(err))
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
		walletCommon.EthereumHoodi,
		walletCommon.EthereumSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalMainnet)
	case walletCommon.OptimismMainnet,
		walletCommon.OptimismSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalOptimism)
	case walletCommon.ArbitrumMainnet,
		walletCommon.ArbitrumSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalArbitrum)
	case walletCommon.BaseMainnet,
		walletCommon.BaseSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalBase)
	case walletCommon.LineaMainnet,
		walletCommon.LineaSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalLinea)
	case walletCommon.UnichainMainnet,
		walletCommon.UnichainSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalUnichain)
	case walletCommon.KatanaMainnet,
		walletCommon.KatanaBokuto:
		ticker = time.NewTicker(newBlockCheckIntervalKatana)
	case walletCommon.InkMainnet,
		walletCommon.InkSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalInk)
	case walletCommon.AbstractMainnet,
		walletCommon.AbstractTestnet:
		ticker = time.NewTicker(newBlockCheckIntervalAbstract)
	case walletCommon.ZkSyncMainnet,
		walletCommon.ZkSyncSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalZkSync)
	case walletCommon.SoneiumMainnet,
		walletCommon.SoneiumMinato:
		ticker = time.NewTicker(newBlockCheckIntervalSoneium)
	case walletCommon.ScrollMainnet,
		walletCommon.ScrollSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalScroll)
	case walletCommon.BlastMainnet,
		walletCommon.BlastSepolia:
		ticker = time.NewTicker(newBlockCheckIntervalBlast)
	case walletCommon.RobinhoodMainnet,
		walletCommon.RobinhoodTestnet:
		ticker = time.NewTicker(newBlockCheckIntervalRobinhood)
	case walletCommon.BSCMainnet,
		walletCommon.BSCTestnet:
		ticker = time.NewTicker(newBlockCheckIntervalBSC)
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
					r.logger.Error("subscribeForUdates: failed to get block number",
						zap.Uint64("chainId", chainID),
						zap.Error(err))
					r.sendUpdatesError(err)
					continue
				}

				val, ok := r.clientsForUpdatesPerChains.Load(chainID)
				if !ok {
					r.logger.Error("subscribeForUdates: failed to load last block details",
						zap.Uint64("chainId", chainID))
					r.sendUpdatesError(err)
					continue
				}

				flbLoaded, ok := val.(fetchingLastBlock)
				if !ok {
					r.logger.Error("subscribeForUdates: failed to cast last block details",
						zap.Uint64("chainId", chainID))
					r.sendUpdatesError(err)
					continue
				}

				if blockNumber > flbLoaded.lastBlock {
					r.logger.Debug("subscribeForUdates: new block detected, refreshing fees",
						zap.Uint64("chainId", chainID),
						zap.Uint64("blockNumber", blockNumber),
						zap.Uint64("prevBlock", flbLoaded.lastBlock))
					flbLoaded.lastBlock = blockNumber
					r.clientsForUpdatesPerChains.Store(chainID, flbLoaded)

					fees, noBaseFee, noPriorityFee, err := r.feesManager.SuggestedFees(ctx, chainID, address)
					if err != nil {
						r.logger.Error("subscribeForUdates: failed to get suggested fees",
							zap.Uint64("chainId", chainID),
							zap.Error(err))
						r.sendUpdatesError(err)
						continue
					}

					r.activeRoutesMutex.Lock()
					if r.activeRoutes != nil && r.activeRoutes.Route != nil && len(r.activeRoutes.Route) > 0 {
						r.logger.Debug("subscribeForUdates: re-evaluating active route paths",
							zap.Uint64("chainId", chainID),
							zap.Int("paths", len(r.activeRoutes.Route)))
						usedNonces := make(map[uint64]uint64)
						for _, path := range r.activeRoutes.Route {
							err = r.evaluateAndUpdatePathDetails(ctx, path, fees, usedNonces, noBaseFee, noPriorityFee, false, 0)
							if err != nil {
								break
							}
						}
						if err != nil {
							r.logger.Error("subscribeForUdates: failed to recalculate fees for path",
								zap.Uint64("chainId", chainID),
								zap.Error(err))
							r.activeRoutesMutex.Unlock()
							r.sendUpdatesError(err)
							continue
						}

						err = r.checkBalancesForTheBestRoute(r.activeRoutes.Route)
						if err != nil {
							r.logger.Error("subscribeForUdates: balance check failed after fee refresh",
								zap.Uint64("chainId", chainID),
								zap.Error(err))
							r.activeRoutesMutex.Unlock()
							r.sendUpdatesError(err)
							continue
						}
					}
					r.activeRoutesMutex.Unlock()

					r.sendUpdatesError(err)
				}
			case <-flb.closeCh:
				r.logger.Debug("subscribeForUdates: stopping update loop", zap.Uint64("chainId", chainID))
				ticker.Stop()
				cancelCtx()
				return
			}
		}
	}()
	return nil
}

func (r *Router) sendUpdatesError(err error) {
	r.lastInputParamsMutex.Lock()
	uuid := r.lastInputParams.Uuid
	r.lastInputParamsMutex.Unlock()

	r.logger.Debug("sendUpdatesError: emitting update result",
		zap.String("uuid", uuid),
		zap.Bool("hasError", err != nil))

	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()

	sendRouterResult(uuid, r.activeRoutes, err)
}

func (r *Router) startTimeoutForUpdates(closeCh chan struct{}, timeout time.Duration) {
	r.logger.Debug("startTimeoutForUpdates: starting update timeout", zap.Duration("timeout", timeout))
	dedlineTicker := time.NewTicker(timeout)
	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case <-dedlineTicker.C:
				r.logger.Info("startTimeoutForUpdates: deadline reached, unsubscribing from fee updates",
					zap.Duration("timeout", timeout))
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
	r.logger.Debug("unsubscribeFeesUpdateAccrossAllChains: unsubscribing from fee updates on all chains")
	r.clientsForUpdatesPerChains.Range(func(key, value interface{}) bool {
		flb, ok := value.(fetchingLastBlock)
		if !ok {
			r.logger.Error("unsubscribeFeesUpdateAccrossAllChains: failed to cast fetchingLastBlock",
				zap.Any("chainId", key))
			return false
		}

		r.logger.Debug("unsubscribeFeesUpdateAccrossAllChains: closing update channel",
			zap.Any("chainId", key))
		close(flb.closeCh)
		r.clientsForUpdatesPerChains.Delete(key)
		return true
	})
}
