package router

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/requests"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func bigHex(v int64) *hexutil.Big { return (*hexutil.Big)(big.NewInt(v)) }

type gasEstimator struct {
	gas   uint64
	err   error
	calls int
}

func (e *gasEstimator) estimate() (uint64, error) {
	e.calls++
	return e.gas, e.err
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
		tx, approval := &gasEstimator{gas: 217760}, &gasEstimator{}
		refreshPathGasEstimates(path, now, tx.estimate, approval.estimate, logger)

		require.Equal(t, 1, tx.calls)
		require.Equal(t, 0, approval.calls)
		require.Equal(t, uint64(217760), path.SuggestedTxGasAmount)
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
		tx, approval := &gasEstimator{gas: 999999}, &gasEstimator{gas: 150213}
		refreshPathGasEstimates(path, now, tx.estimate, approval.estimate, logger)

		require.Equal(t, 0, tx.calls)
		require.Equal(t, 1, approval.calls)
		require.Equal(t, uint64(150213), path.SuggestedApprovalGasAmount)
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
