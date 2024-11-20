package router

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/google/uuid"

	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router/fees"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
)

const (
	testBaseFee           = 50000000000
	testGasPrice          = 10000000000
	testPriorityFeeLow    = 1000000000
	testPriorityFeeMedium = 2000000000
	testPriorityFeeHigh   = 3000000000
	testBonderFeeETH      = 150000000000000
	testBonderFeeUSDC     = 10000

	testAmount0Point1ETHInWei = 100000000000000000
	testAmount0Point2ETHInWei = 200000000000000000
	testAmount0Point3ETHInWei = 300000000000000000
	testAmount0Point4ETHInWei = 400000000000000000
	testAmount0Point5ETHInWei = 500000000000000000
	testAmount0Point6ETHInWei = 600000000000000000
	testAmount0Point8ETHInWei = 800000000000000000
	testAmount1ETHInWei       = 1000000000000000000
	testAmount2ETHInWei       = 2000000000000000000
	testAmount3ETHInWei       = 3000000000000000000
	testAmount5ETHInWei       = 5000000000000000000

	testAmount1USDC   = 1000000
	testAmount100USDC = 100000000

	testApprovalGasEstimation = 1000
	testApprovalL1Fee         = 100000000000

	stageName = "test"
)

var (
	testEstimationMap = map[string]requests.Estimation{
		pathProcessorCommon.ProcessorTransferName:  {Value: uint64(1000), Err: nil},
		pathProcessorCommon.ProcessorBridgeHopName: {Value: uint64(5000), Err: nil},
	}

	testBBonderFeeMap = map[string]*big.Int{
		walletCommon.EthSymbol:  big.NewInt(testBonderFeeETH),
		walletCommon.UsdcSymbol: big.NewInt(testBonderFeeUSDC),
	}

	testTokenPrices = map[string]float64{
		walletCommon.EthSymbol:  2000,
		walletCommon.UsdcSymbol: 1,
	}

	testSuggestedFees = &fees.SuggestedFees{
		GasPrice:             big.NewInt(testGasPrice),
		BaseFee:              big.NewInt(testBaseFee),
		MaxPriorityFeePerGas: big.NewInt(testPriorityFeeLow),
		MaxFeesLevels: &fees.MaxFeesLevels{
			Low:    (*hexutil.Big)(big.NewInt(testPriorityFeeLow)),
			Medium: (*hexutil.Big)(big.NewInt(testPriorityFeeMedium)),
			High:   (*hexutil.Big)(big.NewInt(testPriorityFeeHigh)),
		},
		EIP1559Enabled: false,
	}

	testBalanceMapPerChain = map[string]*big.Int{
		makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC),
		makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC),
		makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC),
	}
)

var mainnet = params.Network{
	ChainID:                walletCommon.EthereumMainnet,
	ChainName:              "Mainnet",
	DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/ethereum/mainnet/", stageName),
	DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/ethereum/mainnet/", stageName),
	DefaultFallbackURL2:    "https://mainnet.infura.io/v3/",
	RPCURL:                 fmt.Sprintf("https://%s.api.status.im/grove/ethereum/mainnet/", stageName),
	FallbackURL:            "https://eth-archival.rpc.grove.city/v1/",
	BlockExplorerURL:       "https://etherscan.io/",
	IconURL:                "network/Network=Ethereum",
	ChainColor:             "#627EEA",
	ShortName:              "eth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  1,
	Enabled:                true,
	RelatedChainID:         walletCommon.EthereumMainnet,
}

var optimism = params.Network{
	ChainID:                walletCommon.OptimismMainnet,
	ChainName:              "Optimism",
	DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/optimism/mainnet/", stageName),
	DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/optimism/mainnet/", stageName),
	DefaultFallbackURL2:    "https://optimism-mainnet.infura.io/v3/",
	RPCURL:                 fmt.Sprintf("https://%s.api.status.im/grove/optimism/mainnet/", stageName),
	FallbackURL:            "https://optimism-archival.rpc.grove.city/v1/",
	BlockExplorerURL:       "https://optimistic.etherscan.io",
	IconURL:                "network/Network=Optimism",
	ChainColor:             "#E90101",
	ShortName:              "oeth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  2,
	Enabled:                true,
	RelatedChainID:         walletCommon.OptimismMainnet,
}

var arbitrum = params.Network{
	ChainID:                walletCommon.ArbitrumMainnet,
	ChainName:              "Arbitrum",
	DefaultRPCURL:          fmt.Sprintf("https://%s.api.status.im/nodefleet/arbitrum/mainnet/", stageName),
	DefaultFallbackURL:     fmt.Sprintf("https://%s.api.status.im/infura/arbitrum/mainnet/", stageName),
	DefaultFallbackURL2:    "https://arbitrum-mainnet.infura.io/v3/",
	RPCURL:                 fmt.Sprintf("https://%s.api.status.im/grove/arbitrum/mainnet/", stageName),
	FallbackURL:            "https://arbitrum-one.rpc.grove.city/v1/",
	IconURL:                "network/Network=Arbitrum",
	ChainColor:             "#51D0F0",
	ShortName:              "arb1",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  2,
	Enabled:                true,
	RelatedChainID:         walletCommon.ArbitrumMainnet,
}

var defaultNetworks = []params.Network{
	mainnet,
	optimism,
	arbitrum,
}

type normalTestParams struct {
	name               string
	input              *requests.RouteInputParams
	expectedCandidates routes.Route
	expectedError      *errors.ErrorResponse
}

func getNormalTestParamsList() []normalTestParams {
	return []normalTestParams{
		{
			name: "ETH transfer - Insufficient Funds",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:   testTokenPrices,
					SuggestedFees: testSuggestedFees,
					BalanceMap:    testBalanceMapPerChain,
					EstimationMap: map[string]requests.Estimation{
						pathProcessorCommon.ProcessorTransferName: {
							Value: uint64(0),
							Err:   fmt.Errorf("failed with 50000000 gas: insufficient funds for gas * price + value: address %s have 68251537427723 want 100000000000000", common.HexToAddress("0x1")),
						},
					},
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: &errors.ErrorResponse{
				Code:    errors.GenericErrorCode,
				Details: fmt.Sprintf("failed with 50000000 gas: insufficient funds for gas * price + value: address %s", common.HexToAddress("0x1")),
			},
			expectedCandidates: routes.Route{},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - 0 AmountIn",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(0)),
				TokenID:     walletCommon.EthSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     walletCommon.EthSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - Specific Single ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:            walletCommon.EthSymbol,
				DisabledToChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - Specific Multiple ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:            walletCommon.EthSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Single FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:   testTokenPrices,
					BaseFee:       big.NewInt(testBaseFee),
					SuggestedFees: testSuggestedFees,
					BalanceMap:    testBalanceMapPerChain,

					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Single FromChain - Specific Single ToChain - Same Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Single FromChain - Specific Single ToChain - Different Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - Specific Multiple ToChain - Single Common Chain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - Specific Multiple ToChain - Multiple Common Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - Specific Multiple ToChain - No Common Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - All FromChains Disabled - All ToChains Disabled",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:   testTokenPrices,
					BaseFee:       big.NewInt(testBaseFee),
					SuggestedFees: testSuggestedFees,
					BalanceMap:    testBalanceMapPerChain,

					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrNoBestRouteFound,
			expectedCandidates: routes.Route{},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - Single Chain LockedAmount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - Specific ToChain - Single Chain LockedAmount",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:            walletCommon.EthSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount1ETHInWei - testAmount0Point2ETHInWei - testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)), //(*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)), //(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)), //(*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - Multiple Chains LockedAmount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - All Chains LockedAmount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - All Chains LockedAmount with insufficient amount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point4ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      requests.ErrLockedAmountLessThanSendAmountAllNetworks,
			expectedCandidates: routes.Route{},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - LockedAmount exceeds sending amount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      requests.ErrLockedAmountExceedsTotalSendAmount,
			expectedCandidates: routes.Route{},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:     walletCommon.UsdcSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific Single ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific Multiple ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Single FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Single FromChain - Specific Single ToChain - Same Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Single FromChain - Specific Single ToChain - Different Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - Specific Multiple ToChain - Single Common Chain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - Specific Multiple ToChain - Multiple Common Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - Specific Multiple ToChain - No Common Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - All FromChains Disabled - All ToChains Disabled",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrNoBestRouteFound,
			expectedCandidates: routes.Route{},
		},
		{
			name: "ERC20 transfer - All FromChains - No Locked Amount - Enough Token Balance Across All Chains",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
				TokenID:     walletCommon.UsdcSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - No Specific FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Bridge,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:     walletCommon.UsdcSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - No Specific FromChain - Specific Single ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Bridge,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - No Specific FromChain - Specific Multiple ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Bridge,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Single FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Multiple FromChain - No Specific ToChain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Single FromChain - Specific Single ToChain - Same Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrNoBestRouteFound,
			expectedCandidates: routes.Route{},
		},
		{
			name: "Bridge - Specific Single FromChain - Specific Single ToChain - Different Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Multiple FromChain - Specific Multiple ToChain - Single Common Chain",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrNoBestRouteFound,
			expectedCandidates: routes.Route{},
		},
		{
			name: "Bridge - Specific Multiple FromChain - Specific Multiple ToChain - Multiple Common Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Multiple FromChain - Specific Multiple ToChain - No Common Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - All FromChains Disabled - All ToChains Disabled",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrNoBestRouteFound,
			expectedCandidates: routes.Route{},
		},
		{
			name: "ETH transfer - Not Enough Native Balance",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					TokenPrices:           testTokenPrices,
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: &errors.ErrorResponse{
				Code:    ErrNotEnoughNativeBalance.Code,
				Details: fmt.Sprintf(ErrNotEnoughNativeBalance.Details, walletCommon.EthSymbol, walletCommon.EthereumMainnet),
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Not Enough Native Balance",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(5 * testAmount100USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: &errors.ErrorResponse{
				Code:    ErrNotEnoughTokenBalance.Code,
				Details: fmt.Sprintf(ErrNotEnoughTokenBalance.Details, walletCommon.UsdcSymbol, walletCommon.EthereumMainnet),
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Single FromChain - Specific Single ToChain - Sending Small Amount",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(0.01 * testAmount1USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:           testTokenPrices,
					BaseFee:               big.NewInt(testBaseFee),
					SuggestedFees:         testSuggestedFees,
					BalanceMap:            testBalanceMapPerChain,
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrLowAmountInForHopBridge,
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
	}
}

type noBalanceTestParams struct {
	name               string
	input              *requests.RouteInputParams
	expectedCandidates routes.Route
	expectedBest       routes.Route
	expectedError      *errors.ErrorResponse
}

func getNoBalanceTestParamsList() []noBalanceTestParams {
	return []noBalanceTestParams{
		{
			name: "ERC20 transfer - Specific FromChain - Specific ToChain - Not Enough Token Balance",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:   testTokenPrices,
					SuggestedFees: testSuggestedFees,
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.UsdcSymbol): big.NewInt(0),
					},
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrNoPositiveBalance,
		},
		{
			name: "ERC20 transfer - Specific FromChain - Specific ToChain - Not Enough Native Balance",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:              walletCommon.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:   testTokenPrices,
					SuggestedFees: testSuggestedFees,
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol):  big.NewInt(0),
					},
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: &errors.ErrorResponse{
				Code:    ErrNotEnoughNativeBalance.Code,
				Details: fmt.Sprintf(ErrNotEnoughNativeBalance.Details, walletCommon.EthSymbol, walletCommon.OptimismMainnet),
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:         pathProcessorCommon.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					RequiredTokenBalance:  big.NewInt(testAmount100USDC),
					RequiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific ToChain - Not Enough Token Balance Across All Chains",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:   testTokenPrices,
					SuggestedFees: testSuggestedFees,
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.UsdcSymbol): big.NewInt(0),
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.UsdcSymbol): big.NewInt(0),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.UsdcSymbol): big.NewInt(0),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol):  big.NewInt(0),
					},
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrNoPositiveBalance,
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific ToChain - Enough Token Balance On Arbitrum Chain But Not Enough Native Balance",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:   testTokenPrices,
					SuggestedFees: testSuggestedFees,
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol):  big.NewInt(0),
					},
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: &errors.ErrorResponse{
				Code:    ErrNotEnoughNativeBalance.Code,
				Details: fmt.Sprintf(ErrNotEnoughNativeBalance.Details, walletCommon.EthSymbol, walletCommon.ArbitrumMainnet),
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:         pathProcessorCommon.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					RequiredTokenBalance:  big.NewInt(testAmount100USDC),
					RequiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific ToChain - Enough Token Balance On Arbitrum Chain And Enough Native Balance On Arbitrum Chain",
			input: &requests.RouteInputParams{
				TestnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           sendtype.Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:            walletCommon.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.UsdcSymbol,
						Decimals: 6,
					},
					TokenPrices:   testTokenPrices,
					SuggestedFees: testSuggestedFees,
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol):  big.NewInt(testAmount1ETHInWei),
					},
					EstimationMap:         testEstimationMap,
					BonderFeeMap:          testBBonderFeeMap,
					ApprovalGasEstimation: testApprovalGasEstimation,
					ApprovalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:         pathProcessorCommon.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					RequiredTokenBalance:  big.NewInt(testAmount100USDC),
					RequiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
			expectedBest: routes.Route{
				{
					ProcessorName:    pathProcessorCommon.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
	}
}

type amountOptionsTestParams struct {
	name                  string
	input                 *requests.RouteInputParams
	expectedAmountOptions map[uint64][]amountOption
}

func getAmountOptionsTestParamsList() []amountOptionsTestParams {
	return []amountOptionsTestParams{
		{
			name: "Transfer - Single From Chain - No Locked Amount",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: false,
					},
				},
			},
		},
		{
			name: "Transfer - Single From Chain - Locked Amount To Single Chain Equal Total Amount",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
			},
		},
		{
			name: "Transfer - Multiple From Chains - Locked Amount To Single Chain Is Less Than Total Amount",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount2ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: false,
					},
				},
			},
		},
		{
			name: "Transfer - Multiple From Chains - Locked Amount To Multiple Chains",
			input: &requests.RouteInputParams{
				TestnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             sendtype.Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount2ETHInWei)),
				TokenID:              walletCommon.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
			},
		},
		{
			name: "Transfer - All From Chains - Locked Amount To Multiple Chains Equal Total Amount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount2ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
			},
		},
		{
			name: "Transfer - All From Chains - Locked Amount To Multiple Chains Is Less Than Total Amount",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount5ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
				walletCommon.EthereumMainnet: {
					{
						amount: big.NewInt(testAmount3ETHInWei),
						locked: false,
					},
				},
			},
		},
		{
			name: "Transfer - All From Chain - No Locked Amount - Enough Token Balance If All Chains Are Used",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:     walletCommon.EthSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
					},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount3ETHInWei),
						locked: false,
					},
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: false,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount3ETHInWei),
						locked: false,
					},
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: false,
					},
				},
				walletCommon.EthereumMainnet: {
					{
						amount: big.NewInt(testAmount3ETHInWei),
						locked: false,
					},
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: false,
					},
				},
			},
		},
		{
			name: "Transfer - All From Chain - Locked Amount To Single Chain - Enough Token Balance If All Chains Are Used",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount2ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount3ETHInWei),
					},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount0Point5ETHInWei),
						locked: true,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount2ETHInWei + testAmount0Point5ETHInWei),
						locked: false,
					},
					{
						amount: big.NewInt(testAmount0Point5ETHInWei),
						locked: false,
					},
				},
				walletCommon.EthereumMainnet: {
					{
						amount: big.NewInt(testAmount2ETHInWei + testAmount0Point5ETHInWei),
						locked: false,
					},
					{
						amount: big.NewInt(testAmount2ETHInWei),
						locked: false,
					},
				},
			},
		},
		{
			name: "Transfer - All From Chain - Locked Amount To Multiple Chains - Enough Token Balance If All Chains Are Used",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:     walletCommon.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount2ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount3ETHInWei),
					},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount0Point5ETHInWei),
						locked: true,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei + testAmount0Point5ETHInWei),
						locked: false,
					},
				},
				walletCommon.EthereumMainnet: {
					{
						amount: big.NewInt(testAmount1ETHInWei),
						locked: true,
					},
				},
			},
		},
		{
			name: "Transfer - All From Chain - No Locked Amount - Not Enough Token Balance",
			input: &requests.RouteInputParams{
				TestnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    sendtype.Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount5ETHInWei)),
				TokenID:     walletCommon.EthSymbol,

				TestsMode: true,
				TestParams: &requests.RouterTestParams{
					TokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   walletCommon.EthSymbol,
						Decimals: 18,
					},
					BalanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, walletCommon.EthSymbol): big.NewInt(testAmount1ETHInWei),
					},
				},
			},
			expectedAmountOptions: map[uint64][]amountOption{
				walletCommon.OptimismMainnet: {
					{
						amount: big.NewInt(testAmount5ETHInWei),
						locked: false,
					},
				},
				walletCommon.ArbitrumMainnet: {
					{
						amount: big.NewInt(testAmount5ETHInWei),
						locked: false,
					},
				},
				walletCommon.EthereumMainnet: {
					{
						amount: big.NewInt(testAmount5ETHInWei),
						locked: false,
					},
				},
			},
		},
	}
}
