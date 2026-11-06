package router

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/requests"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	mockpathprocessor "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/mock"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func bigHex(v int64) *hexutil.Big { return (*hexutil.Big)(big.NewInt(v)) }

type gasEstimator struct {
	gas   uint64 // buffered limit
	raw   uint64 // node estimate
	err   error
	calls int
}

func (e *gasEstimator) estimate() (gasEstimate, error) {
	e.calls++
	return gasEstimate{estimated: e.raw, limit: e.gas}, e.err
}

func TestRefreshPathGasEstimates(t *testing.T) {
	logger := logutils.ZapLogger().Named("router-test")
	now := time.Unix(1_000_000, 0)
	due := now.Add(-routes.GasReestimateInterval)
	recent := now.Add(-time.Second)

	newPath := func(approvalRequired bool, txEstimatedAt, approvalEstimatedAt time.Time) *routes.Path {
		return &routes.Path{
			ProcessorName:              "Transfer",
			FromChain:                  &params.Network{ChainID: 1},
			ApprovalRequired:           approvalRequired,
			SuggestedTxGasAmount:       22050,
			TxGasAmount:                30000, // e.g. a custom gas amount
			TxGasEstimatedAt:           txEstimatedAt,
			SuggestedApprovalGasAmount: 48300,
			ApprovalGasAmount:          48300,
			ApprovalGasEstimatedAt:     approvalEstimatedAt,
		}
	}

	t.Run("successful re-estimate updates the suggestion and its timestamp only", func(t *testing.T) {
		path := newPath(false, due, time.Time{})
		tx, approval := &gasEstimator{gas: 217760, raw: 207391}, &gasEstimator{}
		refreshPathGasEstimates(path, now, tx.estimate, approval.estimate, logger)

		require.Equal(t, 1, tx.calls)
		require.Equal(t, 0, approval.calls)
		require.Equal(t, uint64(217760), path.SuggestedTxGasAmount)
		require.Equal(t, uint64(207391), path.EstimatedTxGasAmount)
		require.Equal(t, now, path.TxGasEstimatedAt)
		// the tx gas amount follows the suggestion in the next fee evaluation, unless it's custom
		require.Equal(t, uint64(30000), path.TxGasAmount)
	})

	t.Run("not due yet: no estimate call", func(t *testing.T) {
		path := newPath(false, recent, time.Time{})
		tx := &gasEstimator{gas: 217760}
		refreshPathGasEstimates(path, now, tx.estimate, (&gasEstimator{}).estimate, logger)

		require.Equal(t, 0, tx.calls)
		require.Equal(t, uint64(22050), path.SuggestedTxGasAmount)
		require.Equal(t, recent, path.TxGasEstimatedAt)
	})

	t.Run("failed re-estimate keeps the previous estimate and its timestamp", func(t *testing.T) {
		path := newPath(false, due, time.Time{})
		tx := &gasEstimator{err: errors.New("execution reverted")}
		refreshPathGasEstimates(path, now, tx.estimate, (&gasEstimator{}).estimate, logger)

		require.Equal(t, 1, tx.calls)
		require.Equal(t, uint64(22050), path.SuggestedTxGasAmount)
		require.Equal(t, due, path.TxGasEstimatedAt)
	})

	t.Run("approval-pending path re-estimates the approval, never the main tx", func(t *testing.T) {
		path := newPath(true, time.Time{}, due)
		tx, approval := &gasEstimator{gas: 999999}, &gasEstimator{gas: 150213, raw: 143060}
		refreshPathGasEstimates(path, now, tx.estimate, approval.estimate, logger)

		require.Equal(t, 0, tx.calls)
		require.Equal(t, 1, approval.calls)
		require.Equal(t, uint64(150213), path.SuggestedApprovalGasAmount)
		require.Equal(t, uint64(143060), path.EstimatedApprovalGasAmount)
		require.Equal(t, now, path.ApprovalGasEstimatedAt)
		require.Equal(t, uint64(22050), path.SuggestedTxGasAmount)
		require.True(t, path.TxGasEstimatedAt.IsZero())
	})

	t.Run("failed approval re-estimate keeps the previous estimate", func(t *testing.T) {
		path := newPath(true, time.Time{}, due)
		approval := &gasEstimator{err: errors.New("rpc down")}
		refreshPathGasEstimates(path, now, (&gasEstimator{}).estimate, approval.estimate, logger)

		require.Equal(t, uint64(48300), path.SuggestedApprovalGasAmount)
		require.Equal(t, due, path.ApprovalGasEstimatedAt)
	})
}

func TestEstimationAmountIn(t *testing.T) {
	// a plain native transfer is estimated with the value it will be signed with
	plain := &routes.Path{AmountIn: bigHex(900), AmountInOriginal: bigHex(1000)}
	require.Equal(t, big.NewInt(900), estimationAmountIn(plain))

	// packed data encodes the built-for amount; keep the two consistent
	packed := &routes.Path{AmountIn: bigHex(900), AmountInOriginal: bigHex(1000), TxPackedData: []byte{0x01}}
	require.Equal(t, big.NewInt(1000), estimationAmountIn(packed))

	// a copy, so a processor can't change the path's amount
	got := estimationAmountIn(plain)
	got.SetInt64(1)
	require.Equal(t, big.NewInt(900), plain.AmountIn.ToInt())
}

func TestFullBalanceSendSubtractsFeesOnce(t *testing.T) {
	const balance = 1_000_000

	nativeToken := &tokentypes.Token{Token: &types.Token{Symbol: "ETH", Address: common.Address{}}}
	path := &routes.Path{
		RouterInputParamsUuid: "uuid",
		ProcessorName:         "Transfer",
		FromChain:             &params.Network{ChainID: 1, NativeCurrencySymbol: "ETH"},
		FromToken:             nativeToken,
		AmountIn:              bigHex(balance),
		AmountInOriginal:      bigHex(balance),
		SubtractFees:          true,
		TxL1Fee:               bigHex(0),
		ApprovalFee:           bigHex(0),
		ApprovalL1Fee:         bigHex(0),
	}
	r := &Router{
		logger:          logutils.ZapLogger().Named("router-test"),
		lastInputParams: &requests.RouteInputParams{Uuid: "uuid", AmountIn: bigHex(balance)},
	}
	r.SetTestBalanceMap(map[string]*big.Int{makeBalanceKey(1, "ETH"): big.NewInt(balance)})
	// the send details report the input amount as the amount sent
	requireInputAmountIn := func(expected int64) {
		require.Equal(t, big.NewInt(expected), r.lastInputParams.AmountIn.ToInt())
	}

	// one fee refresh: evaluate path details, then check balances and rebuild the amount
	refresh := func(txFee int64) error {
		path.TxFee = bigHex(txFee)
		path.TxTotalFee = bigHex(txFee)
		path.RequiredNativeBalance = big.NewInt(balance)
		return r.checkBalancesForRouteAndAdjustAmountIn(routes.Route{path})
	}

	for i := 0; i < 5; i++ {
		require.NoError(t, refresh(21_000))
		require.Equal(t, big.NewInt(balance-21_000), path.AmountIn.ToInt(), "refresh %d", i)
	}

	// a fee increase reduces the amount once, by the difference
	require.NoError(t, refresh(204_600))
	require.Equal(t, big.NewInt(balance-204_600), path.AmountIn.ToInt())
	requireInputAmountIn(balance - 204_600)
	require.NoError(t, refresh(204_600))
	require.Equal(t, big.NewInt(balance-204_600), path.AmountIn.ToInt())
	requireInputAmountIn(balance - 204_600)

	// ...and a decrease gives it back
	require.NoError(t, refresh(21_000))
	require.Equal(t, big.NewInt(balance-21_000), path.AmountIn.ToInt())
	requireInputAmountIn(balance - 21_000)

	// the built-for amount itself never moves
	require.Equal(t, big.NewInt(balance), path.AmountInOriginal.ToInt())

	// fees that eat the whole balance surface as the insufficient native balance error
	err := refresh(balance)
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrNotEnoughNativeBalance.Code)
	require.Equal(t, big.NewInt(balance-21_000), path.AmountIn.ToInt())
}

func TestRefreshStaleGasEstimates(t *testing.T) {
	logger := logutils.ZapLogger().Named("router-test")
	now := time.Unix(1_000_000, 0)
	stale := now.Add(-routes.GasEstimateMaxAge - time.Second)
	fresh := now.Add(-time.Second)

	newPath := func(processor string, approvalRequired bool, txEstimatedAt, approvalEstimatedAt time.Time) *routes.Path {
		return &routes.Path{
			ProcessorName:              processor,
			FromChain:                  &params.Network{ChainID: 1},
			ApprovalRequired:           approvalRequired,
			SuggestedTxGasAmount:       157500,
			EstimatedTxGasAmount:       150000,
			TxGasAmount:                157500,
			TxGasEstimatedAt:           txEstimatedAt,
			SuggestedApprovalGasAmount: 48300,
			EstimatedApprovalGasAmount: 46000,
			ApprovalGasAmount:          48300,
			ApprovalGasEstimatedAt:     approvalEstimatedAt,
		}
	}
	noApprovalPlaced := func(string) bool { return false }
	approvalPlaced := func(string) bool { return true }

	type estimators struct{ tx, approval *gasEstimator }
	run := func(route routes.Route, placed func(string) bool, e estimators) {
		refreshStaleGasEstimates(route, placed, now,
			func(*routes.Path) gasEstimatorFunc { return e.tx.estimate },
			func(*routes.Path) gasEstimatorFunc { return e.approval.estimate },
			logger)
	}

	t.Run("swap signed a while after its approval was mined: the main tx is re-estimated", func(t *testing.T) {
		// re-evaluated when the approval was mined, then the user waited
		path := newPath("Paraswap", true, stale, stale)
		e := estimators{tx: &gasEstimator{gas: 163800, raw: 156000}, approval: &gasEstimator{}}
		run(routes.Route{path}, approvalPlaced, e)

		require.Equal(t, 1, e.tx.calls)
		require.Equal(t, 0, e.approval.calls, "a placed approval is never estimated again")
		require.Equal(t, uint64(163800), path.SuggestedTxGasAmount)
		require.Equal(t, uint64(156000), path.EstimatedTxGasAmount)
		require.Equal(t, now, path.TxGasEstimatedAt)
		// the confirmed gas is kept; the build compares it with the new estimate
		require.Equal(t, uint64(157500), path.TxGasAmount)
	})

	t.Run("approval pending: only the approval is re-estimated", func(t *testing.T) {
		path := newPath("Paraswap", true, time.Time{}, stale)
		e := estimators{tx: &gasEstimator{}, approval: &gasEstimator{gas: 50400, raw: 48000}}
		run(routes.Route{path}, noApprovalPlaced, e)

		require.Equal(t, 0, e.tx.calls)
		require.Equal(t, 1, e.approval.calls)
		require.Equal(t, uint64(48000), path.EstimatedApprovalGasAmount)
		require.Equal(t, now, path.ApprovalGasEstimatedAt)
		require.Equal(t, uint64(48300), path.ApprovalGasAmount)
	})

	t.Run("fresh estimates aren't re-estimated", func(t *testing.T) {
		route := routes.Route{newPath("Transfer", false, fresh, time.Time{}), newPath("Paraswap", true, time.Time{}, fresh)}
		e := estimators{tx: &gasEstimator{}, approval: &gasEstimator{}}
		run(route, noApprovalPlaced, e)

		require.Equal(t, 0, e.tx.calls)
		require.Equal(t, 0, e.approval.calls)
	})

	t.Run("full-balance send whose value changed since the estimate is re-estimated for it", func(t *testing.T) {
		path := newPath("Transfer", false, fresh, time.Time{})
		path.TxGasEstimatedValue = bigHex(974_800)
		path.AmountIn = bigHex(924_400)
		e := estimators{tx: &gasEstimator{gas: 63000, raw: 60000}, approval: &gasEstimator{}}
		run(routes.Route{path}, noApprovalPlaced, e)

		require.Equal(t, 1, e.tx.calls)
		require.Equal(t, uint64(60000), path.EstimatedTxGasAmount)
		require.Equal(t, big.NewInt(924_400), path.TxGasEstimatedValue.ToInt())
		require.True(t, path.GasEstimateFresh(false, now))
	})

	t.Run("a failed re-estimate leaves the estimate stale", func(t *testing.T) {
		path := newPath("Transfer", false, stale, time.Time{})
		e := estimators{tx: &gasEstimator{err: errors.New("rpc down")}, approval: &gasEstimator{}}
		run(routes.Route{path}, noApprovalPlaced, e)

		require.Equal(t, 1, e.tx.calls)
		require.Equal(t, stale, path.TxGasEstimatedAt)
		require.Equal(t, uint64(150000), path.EstimatedTxGasAmount)
	})
}

func TestMainTxGasEstimatedAt(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	require.Equal(t, now, mainTxGasEstimatedAt(false, 150000, now))
	require.Equal(t, now, mainTxGasEstimatedAt(true, 150000, now), "estimated before the approval was mined")
	require.True(t, mainTxGasEstimatedAt(true, 0, now).IsZero(), "a stand-in is no estimate")
}

// The build's route copy must carry the re-estimated gas.
func TestGetBestRouteForBuildCarriesRefreshedEstimates(t *testing.T) {
	now := time.Now()
	stale := now.Add(-routes.GasEstimateMaxAge - time.Second)
	approvalPlaced := func(string) bool { return true }

	newRouter := func(t *testing.T, estimated uint64) (*Router, *routes.Path) {
		ctrl := gomock.NewController(t)
		processor := mockpathprocessor.NewMockPathProcessor(ctrl)
		processor.EXPECT().EstimateGas(gomock.Any(), []byte{0xab}).Return(estimated, nil).Times(1)

		// a swap re-evaluated when its approval was mined, signed more than 90 s later
		path := &routes.Path{
			RouterInputParamsUuid: "uuid",
			ProcessorName:         "Paraswap",
			FromChain:             &params.Network{ChainID: 1},
			ToChain:               &params.Network{ChainID: 1},
			ApprovalRequired:      true,
			TxPackedData:          []byte{0xab},
			SuggestedTxGasAmount:  157500,
			EstimatedTxGasAmount:  150000,
			TxGasAmount:           157500,
			TxGasEstimatedAt:      stale,
		}
		r := &Router{
			logger:          logutils.ZapLogger().Named("router-test"),
			pathProcessors:  map[string]pathprocessor.PathProcessor{"Paraswap": processor},
			activeRoutes:    &SuggestedRoutes{Uuid: "uuid", Route: routes.Route{path}},
			lastInputParams: &requests.RouteInputParams{Uuid: "uuid", AmountIn: bigHex(1000)},
		}
		return r, path
	}

	t.Run("unchanged estimate: the copy is fresh and still covered by the confirmed gas", func(t *testing.T) {
		r, active := newRouter(t, 150500)
		route, input := r.GetBestRouteForBuild(approvalPlaced, now)

		require.Equal(t, "uuid", input.Uuid)
		require.Len(t, route, 1)
		require.NotSame(t, active, route[0])
		require.True(t, route[0].GasEstimateFresh(false, now))
		require.Equal(t, uint64(150500), route[0].EstimatedTxGasAmount)
		require.Equal(t, uint64(157500), route[0].TxGasAmount)
		require.True(t, active.GasEstimateFresh(false, now))
	})

	t.Run("raised estimate reaches the copy, above the confirmed gas", func(t *testing.T) {
		r, _ := newRouter(t, 160000)
		route, _ := r.GetBestRouteForBuild(approvalPlaced, now)

		require.True(t, route[0].GasEstimateFresh(false, now))
		require.Equal(t, uint64(160000), route[0].EstimatedTxGasAmount)
		require.Less(t, route[0].TxGasAmount, route[0].EstimatedTxGasAmount, "the build refuses it with WT-006")
	})
}
