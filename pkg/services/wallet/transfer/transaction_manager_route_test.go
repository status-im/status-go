package transfer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	pathprocessorcommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	mockpathprocessor "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/mock"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

func TestCheckGasEstimatesFresh(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	fresh := now.Add(-time.Second)
	stale := now.Add(-routes.GasEstimateMaxAge - time.Second)

	newPath := func(processor string, approvalRequired bool, txEstimatedAt, approvalEstimatedAt time.Time) *routes.Path {
		return &routes.Path{
			ProcessorName:          processor,
			FromChain:              &params.Network{ChainID: 1},
			ToChain:                &params.Network{ChainID: 1},
			ApprovalRequired:       approvalRequired,
			TxGasEstimatedAt:       txEstimatedAt,
			ApprovalGasEstimatedAt: approvalEstimatedAt,
		}
	}
	noApprovalPlaced := func(string) bool { return false }
	approvalPlacedFor := func(name string) func(string) bool {
		return func(processor string) bool { return processor == name }
	}

	testCases := []struct {
		name           string
		route          routes.Route
		approvalPlaced func(string) bool
		expectExpired  bool
	}{
		// only fees were refreshed since the route was built, or every gas re-estimate failed
		{"main tx estimate too old", routes.Route{newPath("Transfer", false, stale, time.Time{})}, noApprovalPlaced, true},
		{"main tx never estimated", routes.Route{newPath("Transfer", false, time.Time{}, time.Time{})}, noApprovalPlaced, true},
		{"main tx re-estimated recently", routes.Route{newPath("Transfer", false, fresh, time.Time{})}, noApprovalPlaced, false},
		{"any stale path expires the route", routes.Route{
			newPath("Transfer", false, fresh, time.Time{}),
			newPath("ERC721Transfer", false, stale, time.Time{}),
		}, noApprovalPlaced, true},

		// approval-pending main txs can't be estimated yet and aren't checked
		{"approval-pending: fresh approval, main tx never estimated", routes.Route{newPath("Paraswap", true, time.Time{}, fresh)}, noApprovalPlaced, false},
		{"approval-pending: fresh approval, old main tx fallback", routes.Route{newPath("Hop", true, stale, fresh)}, noApprovalPlaced, false},
		{"approval-pending: stale approval", routes.Route{newPath("Paraswap", true, fresh, stale)}, noApprovalPlaced, true},

		// once the approval is placed the main tx is built, so its estimate is what counts
		{"approval placed: main tx re-evaluated", routes.Route{newPath("Paraswap", true, fresh, stale)}, approvalPlacedFor("Paraswap"), false},
		{"approval placed: main tx estimate too old", routes.Route{newPath("Paraswap", true, stale, fresh)}, approvalPlacedFor("Paraswap"), true},
		{"approval placed for another processor", routes.Route{newPath("Paraswap", true, stale, fresh)}, approvalPlacedFor("LiFi"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stalePath, err := checkGasEstimatesFresh(tc.route, tc.approvalPlaced, now)
			if tc.expectExpired {
				require.ErrorIs(t, err, ErrRouteGasEstimateExpired)
				require.NotNil(t, stalePath)
			} else {
				require.NoError(t, err)
				require.Nil(t, stalePath)
			}
		})
	}
}

func TestCheckGasCoversEstimates(t *testing.T) {
	newPath := func(processor string, approvalRequired bool, txGas, txEstimated, approvalGas, approvalEstimated uint64) *routes.Path {
		return &routes.Path{
			ProcessorName:              processor,
			FromChain:                  &params.Network{ChainID: 1},
			ToChain:                    &params.Network{ChainID: 1},
			ApprovalRequired:           approvalRequired,
			TxGasAmount:                txGas,
			EstimatedTxGasAmount:       txEstimated,
			ApprovalGasAmount:          approvalGas,
			EstimatedApprovalGasAmount: approvalEstimated,
		}
	}
	noApprovalPlaced := func(string) bool { return false }
	approvalPlaced := func(string) bool { return true }

	testCases := []struct {
		name           string
		path           *routes.Path
		approvalPlaced func(string) bool
		expectLow      bool
	}{
		{"suggested gas covers the estimate", newPath("Transfer", false, 157500, 150000, 0, 0), noApprovalPlaced, false},
		{"custom gas equal to the estimate", newPath("Transfer", false, 150000, 150000, 0, 0), noApprovalPlaced, false},
		// e.g. 60,000 set for an ERC-20 send whose recipient emptied their balance since
		{"custom gas overtaken by a refreshed estimate", newPath("Transfer", false, 60000, 154000, 0, 0), noApprovalPlaced, true},
		{"no estimate: nothing to compare", newPath("Transfer", false, 21000, 0, 0, 0), noApprovalPlaced, false},

		{"approval pending: low approval gas", newPath("Paraswap", true, 500000, 0, 40000, 46000), noApprovalPlaced, true},
		{"approval pending: swap main tx isn't built yet", newPath("Paraswap", true, 100000, 156000, 48300, 46000), noApprovalPlaced, false},
		{"approval pending: non-swap main tx is built in the same pass", newPath("Hop", true, 100000, 156000, 48300, 46000), noApprovalPlaced, true},
		{"approval placed: the approval isn't built again", newPath("Paraswap", true, 157500, 150000, 40000, 46000), approvalPlaced, false},
		{"approval placed: confirmed main tx gas below the new estimate", newPath("Paraswap", true, 157500, 160000, 48300, 46000), approvalPlaced, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lowPath, err := checkGasCoversEstimates(routes.Route{tc.path}, tc.approvalPlaced)
			if tc.expectLow {
				require.ErrorIs(t, err, ErrGasBelowEstimate)
				require.Same(t, tc.path, lowPath)
				require.True(t, RouteNeedsReview(err))
			} else {
				require.NoError(t, err)
				require.Nil(t, lowPath)
			}
		})
	}
}

func TestRouteNeedsReview(t *testing.T) {
	require.True(t, RouteNeedsReview(ErrRouteGasEstimateExpired))
	require.True(t, RouteNeedsReview(ErrGasBelowEstimate))
	require.False(t, RouteNeedsReview(ErrNoRoute))
	require.False(t, RouteNeedsReview(nil))
}

// A swap whose approval was mined and whose main tx was then re-evaluated; the user signs it later.
func TestBuildSwapMainTxAfterApproval(t *testing.T) {
	tm, _ := setupTestSuite(t) // the transactor mock fails the test if an approval is built
	ctrl := gomock.NewController(t)
	processor := mockpathprocessor.NewMockPathProcessor(ctrl)
	processors := map[string]pathprocessor.PathProcessor{pathprocessorcommon.ProcessorSwapParaswapName: processor}

	nonce := hexutil.Uint64(8)
	path := &routes.Path{
		ProcessorName:        pathprocessorcommon.ProcessorSwapParaswapName,
		FromChain:            &params.Network{ChainID: 1, EIP1559Enabled: true},
		ToChain:              &params.Network{ChainID: 1},
		ApprovalRequired:     true,
		TxNonce:              &nonce,
		TxGasAmount:          157500,
		EstimatedTxGasAmount: 150000,
	}
	route := routes.Route{path}
	tm.routerTransactions = []*wallettypes.RouterTransactionDetails{{
		RouterPath:     path,
		ApprovalTxData: &wallettypes.TransactionData{SentHash: types.Hash{1}},
	}}
	inputParams := &pathprocessor.ProcessorInputParams{FromAddr: common.Address(tm.accountsDB.(*dummyAccountsStorage).account.Address)}
	now := time.Now()

	// 91 s after the re-evaluation, and the build-time re-estimate failed: review, keep the approval
	path.TxGasEstimatedAt = now.Add(-routes.GasEstimateMaxAge - time.Second)
	_, _, _, err := tm.BuildTransactionsFromRoute(route, processors, inputParams)
	require.ErrorIs(t, err, ErrRouteGasEstimateExpired)
	require.True(t, RouteNeedsReview(err))
	require.True(t, tm.ApprovalPlacedForPath(path.ProcessorName))

	// the build-time re-estimate found more gas than the user confirmed: review again
	path.TxGasEstimatedAt = now
	path.EstimatedTxGasAmount = 160000
	_, _, _, err = tm.BuildTransactionsFromRoute(route, processors, inputParams)
	require.ErrorIs(t, err, ErrGasBelowEstimate)

	// the re-estimate still fits the confirmed gas: only the main tx is built, with that gas
	path.EstimatedTxGasAmount = 151000
	processor.EXPECT().BuildTransactionV2(gomock.Any(), int64(-1)).DoAndReturn(
		func(sendArgs *wallettypes.SendTxArgs, _ int64) (*gethtypes.Transaction, uint64, error) {
			require.Equal(t, uint64(157500), uint64(*sendArgs.Gas))
			return gethtypes.NewTx(&gethtypes.DynamicFeeTx{Nonce: uint64(nonce), Gas: uint64(*sendArgs.Gas)}), uint64(nonce), nil
		})
	signingDetails, _, _, err := tm.BuildTransactionsFromRoute(route, processors, inputParams)
	require.NoError(t, err)
	require.Len(t, signingDetails.Hashes, 1)
	require.Equal(t, signingDetails.Hashes[0], tm.routerTransactions[0].TxData.HashToSign)
	require.Equal(t, types.Hash{1}, tm.routerTransactions[0].ApprovalTxData.SentHash, "the placed approval is untouched")
}
