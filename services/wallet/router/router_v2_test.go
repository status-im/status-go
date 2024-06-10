package router

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBaseFee           = 50000000000
	testGasPrice          = 10000000000
	testPriorityFeeLow    = 1000000000
	testPriorityFeeMedium = 2000000000
	testPriorityFeeHigh   = 3000000000
	testBonderFeeETH      = 150000000000000
	testBonderFeeUSDC     = 10000

	testAmount0Point2ETHInWei = 200000000000000000
	testAmount0Point3ETHInWei = 300000000000000000
	testAmount0Point4ETHInWei = 400000000000000000
	testAmount0Point5ETHInWei = 500000000000000000
	testAmount0Point6ETHInWei = 600000000000000000
	testAmount0Point8ETHInWei = 800000000000000000
	testAmount1ETHInWei       = 1000000000000000000
	testAmount2ETHInWei       = 2000000000000000000

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
		makeTestBalanceKey(walletCommon.EthereumMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeTestBalanceKey(walletCommon.EthereumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
		makeTestBalanceKey(walletCommon.OptimismMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeTestBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
		makeTestBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount2ETHInWei),
		makeTestBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
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

func setupTestNetworkDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "wallet-router-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupRouter(t *testing.T) (*Router, func()) {
	db, cleanTmpDb := setupTestNetworkDB(t)

	client, _ := rpc.NewClient(nil, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, defaultNetworks, db)

	router := NewRouter(client, nil, nil, nil, nil, nil, nil, nil)

	transfer := pathprocessor.NewTransferProcessor(nil, nil)
	router.AddPathProcessor(transfer)

	erc721Transfer := pathprocessor.NewERC721Processor(nil, nil)
	router.AddPathProcessor(erc721Transfer)

	erc1155Transfer := pathprocessor.NewERC1155Processor(nil, nil)
	router.AddPathProcessor(erc1155Transfer)

	hop := pathprocessor.NewHopBridgeProcessor(nil, nil, nil)
	router.AddPathProcessor(hop)

	paraswap := pathprocessor.NewSwapParaswapProcessor(nil, nil, nil)
	router.AddPathProcessor(paraswap)

	ensRegister := pathprocessor.NewENSReleaseProcessor(nil, nil, nil)
	router.AddPathProcessor(ensRegister)

	ensRelease := pathprocessor.NewENSReleaseProcessor(nil, nil, nil)
	router.AddPathProcessor(ensRelease)

	ensPublicKey := pathprocessor.NewENSPublicKeyProcessor(nil, nil, nil)
	router.AddPathProcessor(ensPublicKey)

	buyStickers := pathprocessor.NewStickersBuyProcessor(nil, nil, nil)
	router.AddPathProcessor(buyStickers)

	return router, cleanTmpDb
}

func TestRouterV2(t *testing.T) {
	router, cleanTmpDb := setupRouter(t)
	defer cleanTmpDb()

	tests := []struct {
		name               string
		input              *RouteInputParams
		expectedCandidates []*PathV2
		expectedError      error
	}{
		{
			name: "ETH transfer - No Specific FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode: false,
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
			name: "Bridge - No Specific FromChain - No Specific ToChain",
			input: &RouteInputParams{
				testnetMode: false,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			routes, err := router.SuggestedRoutesV2(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, routes)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedCandidates), len(routes.Candidates))

				for _, c := range routes.Candidates {
					found := false
					for _, expC := range tt.expectedCandidates {
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
		})
	}
}

func TestNoBalanceForTheBestRouteRouterV2(t *testing.T) {
	router, cleanTmpDb := setupRouter(t)
	defer cleanTmpDb()

	tests := []struct {
		name               string
		input              *RouteInputParams
		expectedCandidates []*PathV2
		expectedBest       []*PathV2
		expectedError      error
	}{
		{
			name: "ERC20 transfer - Specific FromChain - Specific ToChain - Not Enough Token Balance",
			input: &RouteInputParams{
				testnetMode:          false,
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
					tokenPrices:           testTokenPrices,
					suggestedFees:         testSuggestedFees,
					balanceMap:            map[string]*big.Int{},
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
						makeTestBalanceKey(walletCommon.OptimismMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC),
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
					tokenPrices:           testTokenPrices,
					suggestedFees:         testSuggestedFees,
					balanceMap:            map[string]*big.Int{},
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
						makeTestBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
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
						makeTestBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.UsdcSymbol): big.NewInt(testAmount100USDC + testAmount100USDC),
						makeTestBalanceKey(walletCommon.ArbitrumMainnet, pathprocessor.EthSymbol):  big.NewInt(testAmount1ETHInWei),
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			routes, err := router.SuggestedRoutesV2(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.NotNil(t, routes)
				assert.Equal(t, len(tt.expectedCandidates), len(routes.Candidates))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedCandidates), len(routes.Candidates))
				assert.Equal(t, len(tt.expectedBest), len(routes.Best))

				for _, c := range routes.Candidates {
					found := false
					for _, expC := range tt.expectedCandidates {
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

				for _, c := range routes.Best {
					found := false
					for _, expC := range tt.expectedBest {
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
		})
	}
}

func TestValidateInputData(t *testing.T) {
	testCases := []struct {
		name          string
		input         *RouteInputParams
		expectedError error
	}{
		{
			name: "ENSRegister valid data on testnet",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				Username:    "validusername.eth",
				PublicKey:   "validpublickey",
				TokenID:     pathprocessor.SttSymbol,
				testnetMode: true,
			},
			expectedError: nil,
		},
		{
			name: "ENSRegister valid data on mainnet",
			input: &RouteInputParams{
				SendType:  ENSRegister,
				Username:  "validusername.eth",
				PublicKey: "validpublickey",
				TokenID:   pathprocessor.SntSymbol,
			},
			expectedError: nil,
		},
		{
			name: "ENSRegister missing username",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				PublicKey:   "validpublickey",
				TokenID:     pathprocessor.SttSymbol,
				testnetMode: true,
			},
			expectedError: errors.New("username and public key are required for ENSRegister"),
		},
		{
			name: "ENSRegister missing public key",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				Username:    "validusername.eth",
				TokenID:     pathprocessor.SttSymbol,
				testnetMode: true,
			},
			expectedError: errors.New("username and public key are required for ENSRegister"),
		},
		{
			name: "ENSRegister invalid token on testnet",
			input: &RouteInputParams{
				SendType:    ENSRegister,
				Username:    "validusername.eth",
				PublicKey:   "validpublickey",
				TokenID:     "invalidtoken",
				testnetMode: true,
			},
			expectedError: errors.New("only STT is supported for ENSRegister on testnet"),
		},
		{
			name: "ENSRegister invalid token on mainnet",
			input: &RouteInputParams{
				SendType:  ENSRegister,
				Username:  "validusername.eth",
				PublicKey: "validpublickey",
				TokenID:   "invalidtoken",
			},
			expectedError: errors.New("only SNT is supported for ENSRegister"),
		},
		{
			name: "ENSRelease valid data",
			input: &RouteInputParams{
				SendType: ENSRelease,
				Username: "validusername.eth",
			},
			expectedError: nil,
		},
		{
			name: "ENSRelease missing username",
			input: &RouteInputParams{
				SendType: ENSRelease,
			},
			expectedError: errors.New("username is required for ENSRelease"),
		},
		{
			name: "ENSSetPubKey valid data",
			input: &RouteInputParams{
				SendType:  ENSSetPubKey,
				Username:  "validusername.eth",
				PublicKey: "validpublickey",
			},
			expectedError: nil,
		},
		{
			name: "ENSSetPubKey missing username",
			input: &RouteInputParams{
				SendType:  ENSSetPubKey,
				PublicKey: "validpublickey",
			},
			expectedError: errors.New("username and public key are required for ENSSetPubKey"),
		},
		{
			name: "ENSSetPubKey missing public key",
			input: &RouteInputParams{
				SendType: ENSSetPubKey,
				Username: "validusername",
			},
			expectedError: errors.New("username and public key are required for ENSSetPubKey"),
		},
		{
			name: "ENSSetPubKey invalid ENS username",
			input: &RouteInputParams{
				SendType:  ENSSetPubKey,
				Username:  "invalidusername",
				PublicKey: "validpublickey",
			},
			expectedError: errors.New("username and public key are required for ENSSetPubKey"),
		},
		{
			name: "fromLockedAmount with supported network on testnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumSepolia: (*hexutil.Big)(big.NewInt(10)),
				},
				testnetMode: true,
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with supported network on mainnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(10)),
				},
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with supported mainnet network while in test mode",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(10)),
				},
				testnetMode: true,
			},
			expectedError: errors.New("locked amount is not supported for the selected network"),
		},
		{
			name: "fromLockedAmount with unsupported network on testnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					999: (*hexutil.Big)(big.NewInt(10)),
				},
				testnetMode: true,
			},
			expectedError: errors.New("locked amount is not supported for the selected network"),
		},
		{
			name: "fromLockedAmount with unsupported network on mainnet",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					999: (*hexutil.Big)(big.NewInt(10)),
				},
			},
			expectedError: errors.New("locked amount is not supported for the selected network"),
		},
		{
			name: "fromLockedAmount with negative amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(-10)),
				},
			},
			expectedError: errors.New("locked amount must not be negative"),
		},
		{
			name: "fromLockedAmount with zero amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with zero amounts",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectedError: nil,
		},
		{
			name: "fromLockedAmount with all supported networks with zero amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumMainnet: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.OptimismMainnet: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.ArbitrumMainnet: (*hexutil.Big)(big.NewInt(0)),
				},
			},
			expectedError: errors.New("all supported chains are excluded, routing impossible"),
		},
		{
			name: "fromLockedAmount with all supported test networks with zero amount",
			input: &RouteInputParams{
				FromLockedAmount: map[uint64]*hexutil.Big{
					walletCommon.EthereumSepolia: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.OptimismSepolia: (*hexutil.Big)(big.NewInt(0)),
					walletCommon.ArbitrumSepolia: (*hexutil.Big)(big.NewInt(0)),
				},
				testnetMode: true,
			},
			expectedError: errors.New("all supported chains are excluded, routing impossible"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInputData(tc.input)
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError.Error())
			}
		})
	}
}
