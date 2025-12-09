package common

import (
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
)

const (
	HexAddressLength = 42

	StatusDomain = "stateofus.eth"
	EthDomain    = "eth"

	EthSymbol     = "ETH"
	SntSymbol     = "SNT"
	SttSymbol     = "STT"
	UsdcSymbol    = "USDC"
	UsdcSymbolEVM = "USDC (EVM)"
	HopSymbol     = "HOP"
	DaiSymbol     = "DAI"
	BNBSymbol     = "BNB"

	StatusMainnetTokenCrossChainID = "status"
	StatusTestTokenCrossChainID    = "status-test-token"
)

// ProviderID represents the internal ID of a blockchain provider
type ProviderID = string

// Provider IDs
const (
	StatusSmartProxy  = "status-smart-proxy"
	ProxyNodefleet    = "proxy-nodefleet"
	ProxyInfura       = "proxy-infura"
	ProxyGrove        = "proxy-grove"
	SmartProxyAlchemy = "smart-proxy-alchemy"
	Nodefleet         = "nodefleet"
	Infura            = "infura"
	Grove             = "grove"
	Alchemy           = "alchemy"
	DirectInfura      = "direct-infura"
	DirectGrove       = "direct-grove"
	DirectStatus      = "direct-status"
)

type ChainID uint64

const (
	UnknownChainID       uint64 = 0
	EthereumMainnet      uint64 = 1
	EthereumSepolia      uint64 = 11155111
	OptimismMainnet      uint64 = 10
	OptimismSepolia      uint64 = 11155420
	ArbitrumMainnet      uint64 = 42161
	ArbitrumSepolia      uint64 = 421614
	BSCMainnet           uint64 = 56
	BSCTestnet           uint64 = 97
	AnvilMainnet         uint64 = 31337
	BaseMainnet          uint64 = 8453
	BaseSepolia          uint64 = 84532
	StatusNetworkSepolia uint64 = 1660990954
	TestnetChainID       uint64 = 777333
)

var (
	SupportedNetworks = map[uint64]bool{
		EthereumMainnet: true,
		OptimismMainnet: true,
		ArbitrumMainnet: true,
		BaseMainnet:     true,
		BSCMainnet:      true,
	}

	SupportedTestNetworks = map[uint64]bool{
		EthereumSepolia:      true,
		OptimismSepolia:      true,
		ArbitrumSepolia:      true,
		BaseSepolia:          true,
		BSCTestnet:           true,
		StatusNetworkSepolia: true,
		AnvilMainnet:         true,
	}
)

type ContractType byte

const (
	ContractTypeUnknown ContractType = iota
	ContractTypeERC20
	ContractTypeERC721
	ContractTypeERC1155
)

func ZeroAddress() common.Address {
	return common.Address{}
}

func ZeroBigIntValue() *big.Int {
	return big.NewInt(0)
}

func ZeroHash() common.Hash {
	return common.Hash{}
}

func (c ChainID) String() string {
	return strconv.FormatUint(uint64(c), 10)
}

func (c ChainID) ToUint() uint64 {
	return uint64(c)
}

func (c ChainID) IsMainnet() bool {
	switch uint64(c) {
	case EthereumMainnet, OptimismMainnet, ArbitrumMainnet, BaseMainnet, BSCMainnet:
		return true
	case EthereumSepolia, OptimismSepolia, ArbitrumSepolia, BaseSepolia, BSCTestnet, StatusNetworkSepolia:
		return false
	case UnknownChainID:
		return false
	}
	return false
}

func AllChainIDs() []ChainID {
	return []ChainID{
		ChainID(EthereumMainnet),
		ChainID(EthereumSepolia),
		ChainID(OptimismMainnet),
		ChainID(OptimismSepolia),
		ChainID(ArbitrumMainnet),
		ChainID(ArbitrumSepolia),
		ChainID(BaseMainnet),
		ChainID(BaseSepolia),
		ChainID(StatusNetworkSepolia),
		ChainID(BSCMainnet),
		ChainID(BSCTestnet),
		ChainID(TestnetChainID),
		ChainID(AnvilMainnet),
	}
}

func AllChainIDsAsUint64() []uint64 {
	chains := make([]uint64, 0)
	for _, chain := range AllChainIDs() {
		chains = append(chains, uint64(chain))
	}
	return chains
}

func IsSupportedChainID(chainID uint64) bool {
	return SupportedNetworks[chainID] || SupportedTestNetworks[chainID]
}

var AverageBlockDurationForChain = map[ChainID]time.Duration{
	ChainID(UnknownChainID):       time.Duration(12000) * time.Millisecond,
	ChainID(EthereumMainnet):      time.Duration(12000) * time.Millisecond,
	ChainID(EthereumSepolia):      time.Duration(12000) * time.Millisecond,
	ChainID(OptimismMainnet):      time.Duration(2000) * time.Millisecond,
	ChainID(OptimismSepolia):      time.Duration(2000) * time.Millisecond,
	ChainID(ArbitrumMainnet):      time.Duration(250) * time.Millisecond,
	ChainID(ArbitrumSepolia):      time.Duration(250) * time.Millisecond,
	ChainID(BaseMainnet):          time.Duration(2000) * time.Millisecond,
	ChainID(BaseSepolia):          time.Duration(2000) * time.Millisecond,
	ChainID(BSCMainnet):           time.Duration(3000) * time.Millisecond,
	ChainID(BSCTestnet):           time.Duration(3000) * time.Millisecond,
	ChainID(StatusNetworkSepolia): time.Duration(2000) * time.Millisecond,
}

const (
	CommunityTokenListID = "community"

	StatusTokenListID            = "status"            // #nosec G101
	UniswapTokenListID           = "uniswap"           // #nosec G101
	CoingeckoEthereumTokenListID = "coingeckoEthereum" // #nosec G101
	CoingeckoOptimismTokenListID = "coingeckoOptimism" // #nosec G101
	CoingeckoArbitrumTokenListID = "coingeckoArbitrum" // #nosec G101
	CoingeckoBSCTokenListID      = "coingeckoBsc"      // #nosec G101
	CoingeckoBaseTokenListID     = "coingeckoBase"     // #nosec G101
)

// ETH
var ethAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	OptimismMainnet:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ArbitrumMainnet:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BaseMainnet:          common.HexToAddress("0x0000000000000000000000000000000000000000"),
	EthereumSepolia:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	OptimismSepolia:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ArbitrumSepolia:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BaseSepolia:          common.HexToAddress("0x0000000000000000000000000000000000000000"),
	StatusNetworkSepolia: common.HexToAddress("0x0000000000000000000000000000000000000000"),
}

// BNB
var bnbAddressesByChainID = map[uint64]common.Address{
	BSCMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BSCTestnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
}

// DAI
var daiAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet: common.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f"),
	BSCMainnet:      common.HexToAddress("0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3"),
	BaseMainnet:     common.HexToAddress("0x50c5725949a6f0c72e6c4a641f24049a917db0cb"),
	ArbitrumMainnet: common.HexToAddress("0xda10009cbd5d07dd0cecc66161fc93d7c9000da1"),
	EthereumSepolia: common.HexToAddress("0x3e622317f8c93f7328350cf0b56d9ed4c620c5d6"),
}

// USDC (EVM)
var usdcEVMAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet:      common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
	OptimismMainnet:      common.HexToAddress("0x0b2c639c533813f4aa9d7837caf62653d097ff85"),
	ArbitrumMainnet:      common.HexToAddress("0xaf88d065e77c8cc2239327c5edb3a432268e5831"),
	BaseMainnet:          common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
	EthereumSepolia:      common.HexToAddress("0x1c7d4b196cb0c7b01d743fbc6116a902379c7238"),
	OptimismSepolia:      common.HexToAddress("0x5fd84259d66cd46123540766be93dfe6d43130d7"),
	ArbitrumSepolia:      common.HexToAddress("0x75faf114eafb1bdbe2f0316df893fd58ce46aa4d"),
	BaseSepolia:          common.HexToAddress("0x036cbd53842c5426634e7929541ec2318f3dcf7e"),
	StatusNetworkSepolia: common.HexToAddress("0xc445a18ca49190578dad62fba3048c07efc07ffe"),
}

// USDC (BSC)
var usdcBSCAddressesByChainID = map[uint64]common.Address{
	BSCMainnet: common.HexToAddress("0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"),
}

// SNT
var sntAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet:      common.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
	OptimismMainnet:      common.HexToAddress("0x650af3c15af43dcb218406d30784416d64cfb6b2"),
	ArbitrumMainnet:      common.HexToAddress("0x707f635951193ddafbb40971a0fcaab8a6415160"),
	BaseMainnet:          common.HexToAddress("0x662015ec830df08c0fc45896fab726542e8ac09e"),
	EthereumSepolia:      common.HexToAddress("0xE452027cdEF746c7Cd3DB31CB700428b16cD8E51"),
	OptimismSepolia:      common.HexToAddress("0x0b5dad18b8791ddb24252b433ec4f21f9e6e5ed0"),
	BaseSepolia:          common.HexToAddress("0xfdb3b57944943a7724fcc0520ee2b10659969a06"),
	StatusNetworkSepolia: common.HexToAddress("0x1c3ac2a186c6149ae7cb4d716ebbd0766e4f898a"),
}

func MandatoryTokens() []string {
	allAddresses := make(map[uint64][]common.Address)

	for chainID, address := range ethAddressesByChainID {
		allAddresses[chainID] = append(allAddresses[chainID], address)
	}
	for chainID, address := range bnbAddressesByChainID {
		allAddresses[chainID] = append(allAddresses[chainID], address)
	}
	for chainID, address := range daiAddressesByChainID {
		allAddresses[chainID] = append(allAddresses[chainID], address)
	}
	for chainID, address := range usdcEVMAddressesByChainID {
		allAddresses[chainID] = append(allAddresses[chainID], address)
	}
	for chainID, address := range usdcBSCAddressesByChainID {
		allAddresses[chainID] = append(allAddresses[chainID], address)
	}
	for chainID, address := range sntAddressesByChainID {
		allAddresses[chainID] = append(allAddresses[chainID], address)
	}

	mandatoryTokens := make([]string, 0)
	for chainID, addresses := range allAddresses {
		for _, address := range addresses {
			mandatoryTokens = append(mandatoryTokens, types.TokenKey(chainID, address))
		}
	}
	return mandatoryTokens
}
