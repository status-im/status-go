package transfer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
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
