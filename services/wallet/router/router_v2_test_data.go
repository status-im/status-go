package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/google/uuid"

	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
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
)

var (
	testEstimationMap = map[string]uint64{
		pathprocessor.ProcessorTransferName:  uint64(1000),
		pathprocessor.ProcessorBridgeHopName: uint64(5000),
	}

	testBbonderFeeMap = map[string]*big.Int{
		pathprocessor.EthSymbol:  big.NewInt(testBonderFeeETH),
		pathprocessor.UsdcSymbol: big.NewInt(testBonderFeeUSDC),
	}

	testTokenPrices = map[string]float64{
		pathprocessor.EthSymbol:  2000,
		pathprocessor.UsdcSymbol: 1,
	}

	testSuggestedFees = &SuggestedFees{
		GasPrice:             big.NewInt(testGasPrice),
		BaseFee:              big.NewInt(testBaseFee),
		MaxPriorityFeePerGas: big.NewInt(testPriorityFeeLow),
		MaxFeesLevels: &MaxFeesLevels{
			Low:    (*hexutil.Big)(big.NewInt(testPriorityFeeLow)),
			Medium: (*hexutil.Big)(big.NewInt(testPriorityFeeMedium)),
			High:   (*hexutil.Big)(big.NewInt(testPriorityFeeHigh)),
		},
		EIP1559Enabled: false,
	}

	testBalanceMapPerChain = map[string]*big.Int{
		makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
		makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
		makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
	}
)

var mainnet = params.Network{
	ChainID:                walletCommon.EthereumMainnet,
	ChainName:              "Mainnet",
	RPCURL:                 "https://eth-archival.rpc.grove.city/v1/",
	FallbackURL:            "https://mainnet.infura.io/v3/",
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

var sepolia = params.Network{
	ChainID:                walletCommon.EthereumSepolia,
	ChainName:              "Mainnet",
	RPCURL:                 "https://sepolia-archival.rpc.grove.city/v1/",
	FallbackURL:            "https://sepolia.infura.io/v3/",
	BlockExplorerURL:       "https://sepolia.etherscan.io/",
	IconURL:                "network/Network=Ethereum",
	ChainColor:             "#627EEA",
	ShortName:              "eth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 true,
	Layer:                  1,
	Enabled:                true,
	RelatedChainID:         walletCommon.EthereumMainnet,
}

var optimism = params.Network{
	ChainID:                walletCommon.OptimismMainnet,
	ChainName:              "Optimism",
	RPCURL:                 "https://optimism-mainnet.rpc.grove.city/v1/",
	FallbackURL:            "https://optimism-mainnet.infura.io/v3/",
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

var optimismSepolia = params.Network{
	ChainID:                walletCommon.OptimismSepolia,
	ChainName:              "Optimism",
	RPCURL:                 "https://optimism-sepolia-archival.rpc.grove.city/v1/",
	FallbackURL:            "https://optimism-sepolia.infura.io/v3/",
	BlockExplorerURL:       "https://sepolia-optimism.etherscan.io/",
	IconURL:                "network/Network=Optimism",
	ChainColor:             "#E90101",
	ShortName:              "oeth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 true,
	Layer:                  2,
	Enabled:                false,
	RelatedChainID:         walletCommon.OptimismMainnet,
}

var arbitrum = params.Network{
	ChainID:                walletCommon.ArbitrumMainnet,
	ChainName:              "Arbitrum",
	RPCURL:                 "https://arbitrum-one.rpc.grove.city/v1/",
	FallbackURL:            "https://arbitrum-mainnet.infura.io/v3/",
	BlockExplorerURL:       "https://arbiscan.io/",
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

var arbitrumSepolia = params.Network{
	ChainID:                walletCommon.ArbitrumSepolia,
	ChainName:              "Arbitrum",
	RPCURL:                 "https://arbitrum-sepolia-archival.rpc.grove.city/v1/",
	FallbackURL:            "https://arbitrum-sepolia.infura.io/v3/",
	BlockExplorerURL:       "https://sepolia-explorer.arbitrum.io/",
	IconURL:                "network/Network=Arbitrum",
	ChainColor:             "#51D0F0",
	ShortName:              "arb1",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 true,
	Layer:                  2,
	Enabled:                false,
	RelatedChainID:         walletCommon.ArbitrumMainnet,
}

var defaultNetworks = []params.Network{
	mainnet,
	sepolia,
	optimism,
	optimismSepolia,
	arbitrum,
	arbitrumSepolia,
}

type normalTestParams struct {
	name               string
	input              *RouteInputParams
	expectedCandidates []*PathV2
	expectedError      *errors.ErrorResponse
}

func getNormalTestParamsList() []normalTestParams {
	return []normalTestParams{
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - Specific Single ToChain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:            pathprocessor.EthSymbol,
				DisabledToChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - Specific Multiple ToChain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:            pathprocessor.EthSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Single FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:   testTokenPrices,
					baseFee:       big.NewInt(testBaseFee),
					suggestedFees: testSuggestedFees,
					balanceMap:    testBalanceMapPerChain,

					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Single FromChain - Specific Single ToChain - Same Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Single FromChain - Specific Single ToChain - Different Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - Specific Multiple ToChain - Single Common Chain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - Specific Multiple ToChain - Multiple Common Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - Specific Multiple FromChain - Specific Multiple ToChain - No Common Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - All FromChains Disabled - All ToChains Disabled",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:   testTokenPrices,
					baseFee:       big.NewInt(testBaseFee),
					suggestedFees: testSuggestedFees,
					balanceMap:    testBalanceMapPerChain,

					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - Single Chain LockedAmount",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - Specific ToChain - Single Chain LockedAmount",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:            pathprocessor.EthSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount1ETHInWei - testAmount0Point2ETHInWei - testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)), //(*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)), //(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)), //(*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - Multiple Chains LockedAmount",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - All Chains LockedAmount",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei - testBonderFeeETH)),
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - All Chains LockedAmount with insufficient amount",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point2ETHInWei)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point4ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrLockedAmountLessThanSendAmountAllNetworks,
			expectedCandidates: []*PathV2{},
		},
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain - LockedAmount exceeds sending amount",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point3ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point8ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError:      ErrLockedAmountExceedsTotalSendAmount,
			expectedCandidates: []*PathV2{},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:     pathprocessor.UsdcSymbol,

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific Single ToChain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific Multiple ToChain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Single FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Single FromChain - Specific Single ToChain - Same Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Single FromChain - Specific Single ToChain - Different Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - Specific Multiple ToChain - Single Common Chain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - Specific Multiple ToChain - Multiple Common Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - Specific Multiple FromChain - Specific Multiple ToChain - No Common Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - All FromChains Disabled - All ToChains Disabled",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{},
		},
		{
			name: "ERC20 transfer - All FromChains - No Locked Amount - Enough Token Balance Across All Chains",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
				TokenID:     pathprocessor.UsdcSymbol,

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &mainnet,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &arbitrum,
					ToChain:          &arbitrum,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5 * testAmount100USDC)),
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(0.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					AmountOut:        (*hexutil.Big)(big.NewInt(2.5*testAmount100USDC - testBonderFeeUSDC)),
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - No Specific FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Bridge,
				AddrFrom:    common.HexToAddress("0x1"),
				AddrTo:      common.HexToAddress("0x2"),
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:     pathprocessor.UsdcSymbol,

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - No Specific FromChain - Specific Single ToChain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Bridge,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - No Specific FromChain - Specific Multiple ToChain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Bridge,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Single FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Multiple FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &optimism,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Single FromChain - Specific Single ToChain - Same Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{},
		},
		{
			name: "Bridge - Specific Single FromChain - Specific Single ToChain - Different Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Multiple FromChain - Specific Multiple ToChain - Single Common Chain",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{},
		},
		{
			name: "Bridge - Specific Multiple FromChain - Specific Multiple ToChain - Multiple Common Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &arbitrum,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - Specific Multiple FromChain - Specific Multiple ToChain - No Common Chains",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &mainnet,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "Bridge - All FromChains Disabled - All ToChains Disabled",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Bridge,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:           testTokenPrices,
					baseFee:               big.NewInt(testBaseFee),
					suggestedFees:         testSuggestedFees,
					balanceMap:            testBalanceMapPerChain,
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{},
		},
	}
}

type noBalanceTestParams struct {
	name               string
	input              *RouteInputParams
	expectedCandidates []*PathV2
	expectedBest       []*PathV2
	expectedError      *errors.ErrorResponse
}

func getNoBalanceTestParamsList() []noBalanceTestParams {
	return []noBalanceTestParams{
		{
			name: "ERC20 transfer - Specific FromChain - Specific ToChain - Not Enough Token Balance",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:   testTokenPrices,
					suggestedFees: testSuggestedFees,
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(0),
					},
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrNotEnoughTokenBalance,
			expectedCandidates: []*PathV2{
				{
					ProcessorName:         pathprocessor.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					requiredTokenBalance:  big.NewInt(testAmount100USDC),
					requiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
			},
		},
		{
			name: "ERC20 transfer - Specific FromChain - Specific ToChain - Not Enough Native Balance",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AddrFrom:             common.HexToAddress("0x1"),
				AddrTo:               common.HexToAddress("0x2"),
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:              pathprocessor.UsdcSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				DisabledToChainIDs:   []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:   testTokenPrices,
					suggestedFees: testSuggestedFees,
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
					},
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrNotEnoughNativeBalance,
			expectedCandidates: []*PathV2{
				{
					ProcessorName:         pathprocessor.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					requiredTokenBalance:  big.NewInt(testAmount100USDC),
					requiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific ToChain - Not Enough Token Balance Across All Chains",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:   testTokenPrices,
					suggestedFees: testSuggestedFees,
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.UsdcSymbol): big.NewInt(0),
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(0),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(0),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
					},
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrNotEnoughTokenBalance,
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:    pathprocessor.ProcessorTransferName,
					FromChain:        &optimism,
					ToChain:          &optimism,
					ApprovalRequired: false,
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific ToChain - Enough Token Balance On Arbitrum Chain But Not Enough Native Balance",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:   testTokenPrices,
					suggestedFees: testSuggestedFees,
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol):  big.NewInt(0),
					},
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedError: ErrNotEnoughNativeBalance,
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:         pathprocessor.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					requiredTokenBalance:  big.NewInt(testAmount100USDC),
					requiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
		},
		{
			name: "ERC20 transfer - No Specific FromChain - Specific ToChain - Enough Token Balance On Arbitrum Chain And Enough Native Balance On Arbitrum Chain",
			input: &RouteInputParams{
				testnetMode:        false,
				Uuid:               uuid.NewString(),
				SendType:           Transfer,
				AddrFrom:           common.HexToAddress("0x1"),
				AddrTo:             common.HexToAddress("0x2"),
				AmountIn:           (*hexutil.Big)(big.NewInt(testAmount100USDC)),
				TokenID:            pathprocessor.UsdcSymbol,
				DisabledToChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.UsdcSymbol,
						Decimals: 6,
					},
					tokenPrices:   testTokenPrices,
					suggestedFees: testSuggestedFees,
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount1ETHInWei),
					},
					estimationMap:         testEstimationMap,
					bonderFeeMap:          testBbonderFeeMap,
					approvalGasEstimation: testApprovalGasEstimation,
					approvalL1Fee:         testApprovalL1Fee,
				},
			},
			expectedCandidates: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &mainnet,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
				{
					ProcessorName:         pathprocessor.ProcessorTransferName,
					FromChain:             &optimism,
					ToChain:               &optimism,
					ApprovalRequired:      false,
					requiredTokenBalance:  big.NewInt(testAmount100USDC),
					requiredNativeBalance: big.NewInt((testBaseFee + testPriorityFeeLow) * testApprovalGasEstimation),
				},
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
					FromChain:        &arbitrum,
					ToChain:          &optimism,
					ApprovalRequired: true,
				},
			},
			expectedBest: []*PathV2{
				{
					ProcessorName:    pathprocessor.ProcessorBridgeHopName,
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
	input                 *RouteInputParams
	expectedAmountOptions map[uint64][]amountOption
}

func getAmountOptionsTestParamsList() []amountOptionsTestParams {
	return []amountOptionsTestParams{
		{
			name: "Transfer - Single From Chain - No Locked Amount",
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{},
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
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{},
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
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount2ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{},
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
			input: &RouteInputParams{
				testnetMode:          false,
				Uuid:                 uuid.NewString(),
				SendType:             Transfer,
				AmountIn:             (*hexutil.Big)(big.NewInt(testAmount2ETHInWei)),
				TokenID:              pathprocessor.EthSymbol,
				DisabledFromChainIDs: []uint64{walletCommon.EthereumMainnet},
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{},
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
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount2ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{},
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
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount5ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{},
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
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
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
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount2ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount3ETHInWei),
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
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount3ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(testAmount0Point5ETHInWei)),
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(testAmount1ETHInWei)),
				},

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount2ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount3ETHInWei),
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
			input: &RouteInputParams{
				testnetMode: false,
				Uuid:        uuid.NewString(),
				SendType:    Transfer,
				AmountIn:    (*hexutil.Big)(big.NewInt(testAmount5ETHInWei)),
				TokenID:     pathprocessor.EthSymbol,

				testsMode: true,
				testParams: &routerTestParams{
					tokenFrom: &token.Token{
						ChainID:  1,
						Symbol:   pathprocessor.EthSymbol,
						Decimals: 18,
					},
					balanceMap: map[string]*big.Int{
						makeBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
						makeBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol): big.NewInt(testAmount1ETHInWei),
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
