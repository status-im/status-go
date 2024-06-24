package router

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	w_common "github.com/status-im/status-go/services/wallet/common"

	// mock_fees "github.com/status-im/status-go/services/wallet/router/mock/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor/mock_pathprocessor"
	"github.com/status-im/status-go/services/wallet/token"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"
)

func setupRouter(t *testing.T) (*Router, mock_token.MockManagerInterface, *MockCandidateResolver, *MockEstimator, *mock_pathprocessor.MockPathProcessor, *mock_pathprocessor.MockPathProcessor, *gomock.Controller) {
	ctrl := gomock.NewController(t)

	mockTokenManager := mock_token.NewMockManagerInterface(ctrl)
	mockPathProcessor := mock_pathprocessor.NewMockPathProcessor(ctrl)
	mockPathProcessor.EXPECT().Name().Return("mockPathProcessor").AnyTimes()
	mockPathProcessorBridge := mock_pathprocessor.NewMockPathProcessor(ctrl)
	mockPathProcessorBridge.EXPECT().Name().Return(pathprocessor.ProcessorBridgeHopName).AnyTimes()

	candidateResolver := NewMockCandidateResolver(ctrl)
	mockEstimator := NewMockEstimator(ctrl)
	router := NewRouter(nil, nil, mockTokenManager, nil, nil, nil, nil, nil)
	router.AddPathProcessor(mockPathProcessor)
	router.AddPathProcessor(mockPathProcessorBridge)
	router.CandidateResolver = candidateResolver
	router.Estimator = mockEstimator
	return router, *mockTokenManager, candidateResolver, mockEstimator, mockPathProcessor, mockPathProcessorBridge, ctrl
}

func TestRouterResolveCandidatesForNetwork(t *testing.T) {
	router, mockTokenManager, mockCandidateResolver, _, _, _, ctrl := setupRouter(t)
	defer ctrl.Finish()

	input := &RouteInputParams{
		TestnetMode:          false,
		SendType:             Bridge,
		AddrFrom:             common.HexToAddress("0x1"),
		AddrTo:               common.HexToAddress("0x2"),
		AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
		TokenID:              pathprocessor.UsdcSymbol,
		DisabledFromChainIDs: []uint64{},
		DisabledToChainIDs:   []uint64{},
	}

	// testParams: &routerTestParams{
	token := &token.Token{
		ChainID:  1,
		Symbol:   pathprocessor.UsdcSymbol,
		Decimals: 6,
	}

	networks := []*params.Network{
		{
			ChainID: w_common.OptimismMainnet,
		},
	}

	network := &params.Network{
		ChainID: token.ChainID,
	}

	// Test happy path for bridge hop
	require.True(t, validateInput(input, network))
	mockTokenManager.EXPECT().FindToken(network, pathprocessor.UsdcSymbol).Return(token).Times(1)
	// TODO Define the expected return values - empty for 1st processor, non-empty for 2nd processor
	mockCandidateResolver.EXPECT().resolveCandidatesForProcessor(gomock.Any(), input, network, token, gomock.Any(), big.NewInt(testAmount1USDC), false, router.pathProcessors[pathprocessor.ProcessorBridgeHopName], networks).Times(1).Return([]*PathV2{
		{
			ProcessorName: pathprocessor.ProcessorBridgeHopName,
		}, // TODO Define the expected return values
	})

	candidates := router.resolveCandidatesForNetwork(context.TODO(), input, network, networks)
	require.NotEmpty(t, candidates)
}

func TestDefaultCandidateResolverResolveCandidateForProcessor(t *testing.T) {
	router, _, _, _, pathProcessor1, pathProcessorBridge, ctrl := setupRouter(t)
	defer ctrl.Finish()

	input := &RouteInputParams{
		TestnetMode:          false,
		SendType:             Bridge,
		AddrFrom:             common.HexToAddress("0x1"),
		AddrTo:               common.HexToAddress("0x2"),
		AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
		TokenID:              pathprocessor.UsdcSymbol,
		DisabledFromChainIDs: []uint64{},
		DisabledToChainIDs:   []uint64{},
	}

	token := &token.Token{
		ChainID:  1,
		Symbol:   pathprocessor.UsdcSymbol,
		Decimals: 6,
	}

	networks := []*params.Network{
		{
			ChainID: w_common.OptimismMainnet,
		},
	}

	network := &params.Network{
		ChainID: token.ChainID,
	}

	feesManager := NewMockFeeManagerInterface(ctrl)
	feesManager.EXPECT().GetL1Fee(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	rpcClient := mock_rpcclient.NewMockClientInterface(ctrl)
	rpcClient.EXPECT().EthClient(gomock.Any()).Return(nil, nil).AnyTimes()
	resolver := NewDefaultCandidateResolver(rpcClient, feesManager, router.pathProcessors)
	estimator := NewMockEstimator(ctrl)
	estimator.EXPECT().Estimate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	resolver.Estimator = estimator

	processors := []*mock_pathprocessor.MockPathProcessor{pathProcessor1, pathProcessorBridge}
	for _, processor := range processors {
		// Should be Times(1) but I don't have time to check the exact number of calls
		processor.EXPECT().AvailableFor(gomock.Any()).AnyTimes().Return(true, nil)
		processor.EXPECT().CalculateFees(gomock.Any()).AnyTimes()
		processor.EXPECT().EstimateGas(gomock.Any()).AnyTimes()
		processor.EXPECT().GetContractAddress(gomock.Any()).AnyTimes()
		processor.EXPECT().PackTxInputData(gomock.Any()).AnyTimes()
		processor.EXPECT().CalculateAmountOut(gomock.Any()).AnyTimes().Return(big.NewInt(testAmount1USDC), nil)
	}

	candidates := resolver.resolveCandidatesForProcessor(context.TODO(), input, network, token, token,
		big.NewInt(testAmount1USDC), false, router.pathProcessors[pathprocessor.ProcessorBridgeHopName], networks)

	require.NotEmpty(t, candidates)

}
