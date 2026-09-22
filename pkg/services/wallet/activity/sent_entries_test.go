package activity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
)

func TestGetSentActivityTypeForSwapBridgeProcessors(t *testing.T) {
	for _, name := range []string{pathProcessorCommon.ProcessorLiFiName, pathProcessorCommon.ProcessorRelayName} {
		sameChain := &routes.Path{
			ProcessorName: name,
			FromChain:     &params.Network{ChainID: 1},
			ToChain:       &params.Network{ChainID: 1},
		}
		require.Equal(t, ac.SwapAT, getSentActivityType(sameChain, false), name)

		crossChain := &routes.Path{
			ProcessorName: name,
			FromChain:     &params.Network{ChainID: 1},
			ToChain:       &params.Network{ChainID: 10},
		}
		require.Equal(t, ac.BridgeAT, getSentActivityType(crossChain, false), name)
		require.Equal(t, ac.ApproveAT, getSentActivityType(crossChain, true), name)
	}
}
