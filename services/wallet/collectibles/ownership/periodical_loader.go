package ownership

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/pkg/pubsub"
	walletCommon "github.com/status-im/status-go/services/wallet/common"

	"go.uber.org/zap"
)

type LoaderState = int

const (
	LoaderStateIdle LoaderState = iota + 1
	LoaderStateDelayed
	LoaderStateUpdating
	LoaderStateError
	LoaderStateNotAvailable
)

type PeriodicalLoaderParams struct {
	StartDelay   time.Duration // (Optional) Delay between calling Start and actually starting the process, used to "debounce" fetches triggered by detected transfers
	LoadInterval time.Duration // How often to trigger a Load
	LoaderParams LoaderParams
}

func DefaultPeriodicalLoaderParams() PeriodicalLoaderParams {
	return PeriodicalLoaderParams{
		StartDelay:   10 * time.Second,
		LoadInterval: 60 * time.Minute,
		LoaderParams: DefaultLoaderParams(),
	}
}

type PeriodicalLoader struct {
	chainID walletCommon.ChainID
	account common.Address
	loader  *Loader

	params PeriodicalLoaderParams

	state atomic.Value

	stopCh chan struct{}

	logger *zap.Logger
}

func NewPeriodicalLoader(
	chainID walletCommon.ChainID,
	account common.Address,
	fetcher CollectibleOwnershipFetcher,
	storage CollectibleOwnershipStorage,
	publisher *pubsub.Publisher,
	params PeriodicalLoaderParams,
	logger *zap.Logger,
) *PeriodicalLoader {

	ret := &PeriodicalLoader{
		chainID: chainID,
		account: account,
		loader:  NewLoader(chainID, account, fetcher, storage, publisher, params.LoaderParams, logger),
		params:  params,
		logger:  logger.Named("PeriodicalLoader"),
	}
	ret.state.Store(LoaderStateIdle)
	return ret
}

// Starts (or restarts) the loader, triggering an immediate/delayed load
func (pl *PeriodicalLoader) Start(withDelay bool, waitForInterval bool) {
	pl.logger.Debug("starting periodical loader",
		zap.Uint64("chain", uint64(pl.chainID)),
		zap.String("account", gocommon.TruncateWithDot(pl.account.String())),
		zap.Bool("withDelay", withDelay),
		zap.Bool("waitForInterval", waitForInterval),
	)

	// Cancel previous run, if active
	pl.Stop()
	pl.stopCh = make(chan struct{})

	// Trigger
	go func() {
		defer gocommon.LogOnPanic()

		if withDelay && pl.params.StartDelay.Seconds() > 0 {
			pl.state.Store(LoaderStateDelayed)
			if cancelled := pl.waitFor(pl.params.StartDelay); cancelled {
				pl.state.Store(LoaderStateIdle)
				return
			}
		}

		if !waitForInterval {
			// Trigger immediate Load
			pl.logger.Debug("triggering immediate load",
				zap.Uint64("chain", uint64(pl.chainID)),
				zap.String("account", gocommon.TruncateWithDot(pl.account.String())),
			)
			pl.triggerLoadAsync()
		}

		ticker := time.NewTicker(pl.params.LoadInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Trigger periodic Load
				pl.logger.Debug("triggering periodic load",
					zap.Uint64("chain", uint64(pl.chainID)),
					zap.String("account", gocommon.TruncateWithDot(pl.account.String())),
				)
				pl.triggerLoadAsync()
			case <-pl.stopCh:
				return
			}
		}
	}()
}

func (pl *PeriodicalLoader) Stop() {
	if pl.stopCh != nil {
		close(pl.stopCh)
		pl.stopCh = nil
	}
}

func (pl *PeriodicalLoader) GetState() LoaderState {
	return pl.state.Load().(LoaderState)
}

func (pl *PeriodicalLoader) triggerLoadAsync() {
	go func() {
		defer gocommon.LogOnPanic()
		pl.triggerLoad()
	}()
}

func (pl *PeriodicalLoader) triggerLoad() {
	if pl.GetState() == LoaderStateUpdating {
		// Another load is in progress which should've finished or have been cancelled at this point
		pl.logger.Error("skipping Load because it's already in progress",
			zap.Uint64("chain", uint64(pl.chainID)),
			zap.String("account", gocommon.TruncateWithDot(pl.account.String())),
		)
		return
	}

	pl.state.Store(LoaderStateUpdating)

	var err error
	// Handle error at the end (if any) and set state
	defer func() {
		if err != nil {
			pl.state.Store(LoaderStateError)
			return
		}
		pl.state.Store(LoaderStateIdle)
	}()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn() // Cancel ctx after Load ends
	go func() {
		defer gocommon.LogOnPanic()
		defer cancelFn() // Cancel ctx if PeriodicalLoader is stopped
		for {
			select {
			case <-pl.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	err = pl.Load(ctx)
}

func (pl *PeriodicalLoader) Load(ctx context.Context) (err error) {
	_, err = pl.loader.Load(ctx)
	return
}

func (pl *PeriodicalLoader) waitFor(delay time.Duration) (cancelled bool) {
	delayTimer := time.NewTicker(delay)
	defer delayTimer.Stop()
	select {
	case <-delayTimer.C:
		break
	case <-pl.stopCh:
		return true
	}
	return false
}
