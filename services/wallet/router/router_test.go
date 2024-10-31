package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func amountOptionEqual(a, b amountOption) bool {
	return a.amount.Cmp(b.amount) == 0 && a.locked == b.locked
}

func contains(slice []amountOption, val amountOption) bool {
	for _, item := range slice {
		if amountOptionEqual(item, val) {
			return true
		}
	}
	return false
}

func amountOptionsMapsEqual(map1, map2 map[uint64][]amountOption) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, slice1 := range map1 {
		slice2, ok := map2[key]
		if !ok || len(slice1) != len(slice2) {
			return false
		}

		for _, val1 := range slice1 {
			if !contains(slice2, val1) {
				return false
			}
		}

		for _, val2 := range slice2 {
			if !contains(slice1, val2) {
				return false
			}
		}
	}

	return true
}

func assertPathsEqual(t *testing.T, expected, actual routes.Route) {
	assert.Equal(t, len(expected), len(actual))
	if len(expected) == 0 {
		return
	}

	for _, c := range actual {
		found := false
		for _, expC := range expected {
			if c.ProcessorName == expC.ProcessorName &&
				c.FromChain.ChainID == expC.FromChain.ChainID &&
				c.ToChain.ChainID == expC.ToChain.ChainID &&
				c.ApprovalRequired == expC.ApprovalRequired &&
				(expC.AmountOut == nil || c.AmountOut.ToInt().Cmp(expC.AmountOut.ToInt()) == 0) {
				found = true
				break
			}
		}

		assert.True(t, found)
	}
}

func setupTestNetworkDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "wallet-router-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupRouter(t *testing.T) (*Router, func()) {
	db, cleanTmpDb := setupTestNetworkDB(t)

	config := rpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        defaultNetworks,
		DB:              db,
		WalletFeed:      nil,
		ProviderConfigs: nil,
	}
	client, _ := rpc.NewClient(config)

	router := NewRouter(client, nil, nil, nil, nil, nil, nil)

	transfer := pathprocessor.NewTransferProcessor(nil, nil)
	router.AddPathProcessor(transfer)

	erc721Transfer := pathprocessor.NewERC721Processor(nil, nil)
	router.AddPathProcessor(erc721Transfer)

	erc1155Transfer := pathprocessor.NewERC1155Processor(nil, nil)
	router.AddPathProcessor(erc1155Transfer)

	hop := pathprocessor.NewHopBridgeProcessor(nil, nil, nil, nil)
	router.AddPathProcessor(hop)

	paraswap := pathprocessor.NewSwapParaswapProcessor(nil, nil, nil)
	router.AddPathProcessor(paraswap)

	ensRegister := pathprocessor.NewENSReleaseProcessor(nil, nil, nil)
	router.AddPathProcessor(ensRegister)

	ensRelease := pathprocessor.NewENSReleaseProcessor(nil, nil, nil)
	router.AddPathProcessor(ensRelease)

	ensPublicKey := pathprocessor.NewENSPublicKeyProcessor(nil, nil, nil)
	router.AddPathProcessor(ensPublicKey)

	buyStickers := pathprocessor.NewStickersBuyProcessor(nil, nil)
	router.AddPathProcessor(buyStickers)

	return router, cleanTmpDb
}

type routerSuggestedRoutesEnvelope struct {
	Type   string                          `json:"type"`
	Routes responses.RouterSuggestedRoutes `json:"event"`
}

func setupSignalHandler(t *testing.T) (chan responses.RouterSuggestedRoutes, func()) {
	suggestedRoutesCh := make(chan responses.RouterSuggestedRoutes)
	signalHandler := signal.MobileSignalHandler(func(data []byte) {
		var envelope signal.Envelope
		err := json.Unmarshal(data, &envelope)
		assert.NoError(t, err)
		if envelope.Type == string(signal.SuggestedRoutes) {
			var response routerSuggestedRoutesEnvelope
			err := json.Unmarshal(data, &response)
			assert.NoError(t, err)

			suggestedRoutesCh <- response.Routes
		}
	})
	signal.SetMobileSignalHandler(signalHandler)
	t.Cleanup(signal.ResetMobileSignalHandler)

	closeFn := func() {
		close(suggestedRoutesCh)
	}

	return suggestedRoutesCh, closeFn
}

func TestRouter(t *testing.T) {
	router, cleanTmpDb := setupRouter(t)
	defer cleanTmpDb()

	suggestedRoutesCh, closeSignalHandler := setupSignalHandler(t)
	defer closeSignalHandler()

	tests := getNormalTestParamsList()

	// Test blocking endpoints
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes, err := router.SuggestedRoutes(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				if routes == nil {
					assert.Empty(t, tt.expectedCandidates)
				} else {
					assertPathsEqual(t, tt.expectedCandidates, routes.Candidates)
				}
			} else {
				assert.NoError(t, err)
				assertPathsEqual(t, tt.expectedCandidates, routes.Candidates)
			}
		})
	}

	// Test async endpoints
	for _, tt := range tests {
		router.SuggestedRoutesAsync(tt.input)

		select {
		case asyncRoutes := <-suggestedRoutesCh:
			assert.Equal(t, tt.input.Uuid, asyncRoutes.Uuid)
			assert.Equal(t, tt.expectedError, asyncRoutes.ErrorResponse)
			assertPathsEqual(t, tt.expectedCandidates, asyncRoutes.Candidates)
			break
		case <-time.After(10 * time.Second):
			t.FailNow()
		}
	}
}

func TestNoBalanceForTheBestRouteRouter(t *testing.T) {
	router, cleanTmpDb := setupRouter(t)
	defer cleanTmpDb()

	suggestedRoutesCh, closeSignalHandler := setupSignalHandler(t)
	defer closeSignalHandler()

	tests := getNoBalanceTestParamsList()

	// Test blocking endpoints
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			routes, err := router.SuggestedRoutes(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				if tt.expectedError == ErrNoPositiveBalance {
					assert.Nil(t, routes)
				} else {
					assert.NotNil(t, routes)
					assertPathsEqual(t, tt.expectedCandidates, routes.Candidates)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedCandidates), len(routes.Candidates))
				assert.Equal(t, len(tt.expectedBest), len(routes.Best))
				assertPathsEqual(t, tt.expectedCandidates, routes.Candidates)
				assertPathsEqual(t, tt.expectedBest, routes.Best)
			}
		})
	}

	// Test async endpoints
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			router.SuggestedRoutesAsync(tt.input)

			select {
			case asyncRoutes := <-suggestedRoutesCh:
				assert.Equal(t, tt.input.Uuid, asyncRoutes.Uuid)
				assert.Equal(t, tt.expectedError, asyncRoutes.ErrorResponse)
				assertPathsEqual(t, tt.expectedCandidates, asyncRoutes.Candidates)
				if tt.expectedError == nil {
					assertPathsEqual(t, tt.expectedBest, asyncRoutes.Best)
				}
				break
			case <-time.After(10 * time.Second):
				t.FailNow()
			}
		})
	}
}

func TestAmountOptions(t *testing.T) {
	router, cleanTmpDb := setupRouter(t)
	defer cleanTmpDb()

	tests := getAmountOptionsTestParamsList()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			selectedFromChains, _, err := router.getSelectedChains(tt.input)
			assert.NoError(t, err)

			router.SetTestBalanceMap(tt.input.TestParams.BalanceMap)
			amountOptions, err := router.findOptionsForSendingAmount(tt.input, selectedFromChains)
			assert.NoError(t, err)

			assert.Equal(t, len(tt.expectedAmountOptions), len(amountOptions))
			assert.True(t, amountOptionsMapsEqual(tt.expectedAmountOptions, amountOptions))
		})
	}
}
