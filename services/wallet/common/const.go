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
	SmartProxyAlchemy = "smart-proxy-alchemy"
	Nodefleet         = "nodefleet"
	Infura            = "infura"
	Grove             = "grove"
	Alchemy           = "alchemy"
	DirectInfura      = "direct-infura"
	DirectGrove       = "direct-grove"
	DirectCustom      = "direct-custom"
)

type ChainID uint64

const (
	UnknownChainID  uint64 = 0
	EthereumMainnet uint64 = 1
	EthereumSepolia uint64 = 11155111
	OptimismMainnet uint64 = 10
	OptimismSepolia uint64 = 11155420
	ArbitrumMainnet uint64 = 42161
	ArbitrumSepolia uint64 = 421614
	BSCMainnet      uint64 = 56
	BSCTestnet      uint64 = 97
	AnvilMainnet    uint64 = 31337
	BaseMainnet     uint64 = 8453
	BaseSepolia     uint64 = 84532
	LineaMainnet    uint64 = 59144
	LineaSepolia    uint64 = 59141
	UnichainMainnet uint64 = 130
	UnichainSepolia uint64 = 1301
	KatanaMainnet   uint64 = 747474
	KatanaBokuto    uint64 = 737373
	InkMainnet      uint64 = 57073
	InkSepolia      uint64 = 763373
	AbstractMainnet uint64 = 2741
	AbstractTestnet uint64 = 11124
	ZkSyncMainnet   uint64 = 324
	ZkSyncSepolia   uint64 = 300
	SoneiumMainnet  uint64 = 1868
	SoneiumMinato   uint64 = 1946
	ScrollMainnet   uint64 = 534352
	ScrollSepolia   uint64 = 534351
	BlastMainnet    uint64 = 81457
	BlastSepolia    uint64 = 168587773
	EthereumHoodi   uint64 = 560048
	TestnetChainID  uint64 = 777333
)

var (
	SupportedNetworks = map[uint64]bool{
		EthereumMainnet: true,
		OptimismMainnet: true,
		ArbitrumMainnet: true,
		BaseMainnet:     true,
		BSCMainnet:      true,
		LineaMainnet:    true,
		UnichainMainnet: true,
		KatanaMainnet:   true,
		InkMainnet:      true,
		AbstractMainnet: true,
		ZkSyncMainnet:   true,
		SoneiumMainnet:  true,
		ScrollMainnet:   true,
		BlastMainnet:    true,
	}

	SupportedTestNetworks = map[uint64]bool{
		EthereumHoodi:   true,
		EthereumSepolia: true,
		OptimismSepolia: true,
		ArbitrumSepolia: true,
		BaseSepolia:     true,
		LineaSepolia:    true,
		UnichainSepolia: true,
		KatanaBokuto:    true,
		InkSepolia:      true,
		AbstractTestnet: true,
		ZkSyncSepolia:   true,
		SoneiumMinato:   true,
		ScrollSepolia:   true,
		BlastSepolia:    true,
		BSCTestnet:      true,
		AnvilMainnet:    true,
	}
)

type ContractType byte

const (
	ContractTypeUnknown ContractType = iota
	ContractTypeERC20
	ContractTypeERC721
	ContractTypeERC1155
)

// Based on documentation: https://docs.zksync.io/zk-stack/customizations/custom-base-tokens#custom-base-token-setup
func ZkSyncETHTokenAddress() common.Address {
	return common.HexToAddress("0x000000000000000000000000000000000000800a")
}

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
	case EthereumMainnet, OptimismMainnet, ArbitrumMainnet, BaseMainnet, BSCMainnet, LineaMainnet,
		UnichainMainnet, KatanaMainnet, InkMainnet, AbstractMainnet, ZkSyncMainnet,
		SoneiumMainnet, ScrollMainnet, BlastMainnet:
		return true
	case EthereumHoodi, EthereumSepolia, OptimismSepolia, ArbitrumSepolia, BaseSepolia, LineaSepolia,
		UnichainSepolia, KatanaBokuto, InkSepolia, AbstractTestnet, ZkSyncSepolia,
		SoneiumMinato, ScrollSepolia, BlastSepolia,
		BSCTestnet:
		return false
	case UnknownChainID:
		return false
	}
	return false
}

func AllChainIDs() []ChainID {
	return []ChainID{
		ChainID(EthereumMainnet),
		ChainID(EthereumHoodi),
		ChainID(EthereumSepolia),
		ChainID(OptimismMainnet),
		ChainID(OptimismSepolia),
		ChainID(ArbitrumMainnet),
		ChainID(ArbitrumSepolia),
		ChainID(BaseMainnet),
		ChainID(BaseSepolia),
		ChainID(LineaMainnet),
		ChainID(LineaSepolia),
		ChainID(UnichainMainnet),
		ChainID(UnichainSepolia),
		ChainID(KatanaMainnet),
		ChainID(KatanaBokuto),
		ChainID(InkMainnet),
		ChainID(InkSepolia),
		ChainID(AbstractMainnet),
		ChainID(AbstractTestnet),
		ChainID(ZkSyncMainnet),
		ChainID(ZkSyncSepolia),
		ChainID(SoneiumMainnet),
		ChainID(SoneiumMinato),
		ChainID(ScrollMainnet),
		ChainID(ScrollSepolia),
		ChainID(BlastMainnet),
		ChainID(BlastSepolia),
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
	ChainID(UnknownChainID):  time.Duration(12000) * time.Millisecond,
	ChainID(EthereumMainnet): time.Duration(12000) * time.Millisecond,
	ChainID(EthereumHoodi):   time.Duration(12000) * time.Millisecond,
	ChainID(EthereumSepolia): time.Duration(12000) * time.Millisecond,
	ChainID(OptimismMainnet): time.Duration(2000) * time.Millisecond,
	ChainID(OptimismSepolia): time.Duration(2000) * time.Millisecond,
	ChainID(ArbitrumMainnet): time.Duration(250) * time.Millisecond,
	ChainID(ArbitrumSepolia): time.Duration(250) * time.Millisecond,
	ChainID(BaseMainnet):     time.Duration(2000) * time.Millisecond,
	ChainID(BaseSepolia):     time.Duration(2000) * time.Millisecond,
	ChainID(LineaMainnet):    time.Duration(2000) * time.Millisecond,
	ChainID(LineaSepolia):    time.Duration(2000) * time.Millisecond,
	ChainID(UnichainMainnet): time.Duration(2000) * time.Millisecond,
	ChainID(UnichainSepolia): time.Duration(2000) * time.Millisecond,
	ChainID(KatanaMainnet):   time.Duration(2000) * time.Millisecond,
	ChainID(KatanaBokuto):    time.Duration(2000) * time.Millisecond,
	ChainID(InkMainnet):      time.Duration(2000) * time.Millisecond,
	ChainID(InkSepolia):      time.Duration(2000) * time.Millisecond,
	ChainID(AbstractMainnet): time.Duration(2000) * time.Millisecond,
	ChainID(AbstractTestnet): time.Duration(2000) * time.Millisecond,
	ChainID(ZkSyncMainnet):   time.Duration(2000) * time.Millisecond,
	ChainID(ZkSyncSepolia):   time.Duration(2000) * time.Millisecond,
	ChainID(SoneiumMainnet):  time.Duration(2000) * time.Millisecond,
	ChainID(SoneiumMinato):   time.Duration(2000) * time.Millisecond,
	ChainID(ScrollMainnet):   time.Duration(2000) * time.Millisecond,
	ChainID(ScrollSepolia):   time.Duration(2000) * time.Millisecond,
	ChainID(BlastMainnet):    time.Duration(2000) * time.Millisecond,
	ChainID(BlastSepolia):    time.Duration(2000) * time.Millisecond,
	ChainID(BSCMainnet):      time.Duration(3000) * time.Millisecond,
	ChainID(BSCTestnet):      time.Duration(3000) * time.Millisecond,
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
	CoingeckoLineaTokenListID    = "coingeckoLinea"    // #nosec G101
)

// ETH
var ethAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	OptimismMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ArbitrumMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BaseMainnet:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
	LineaMainnet:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
	UnichainMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	KatanaMainnet:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	InkMainnet:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	AbstractMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ZkSyncMainnet:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	SoneiumMainnet:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ScrollMainnet:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BlastMainnet:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
	EthereumHoodi:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	EthereumSepolia: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	OptimismSepolia: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ArbitrumSepolia: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BaseSepolia:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
	LineaSepolia:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
	UnichainSepolia: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	KatanaBokuto:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
	InkSepolia:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
	AbstractTestnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ZkSyncSepolia:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	SoneiumMinato:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	ScrollSepolia:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BlastSepolia:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
}

// BNB
var bnbAddressesByChainID = map[uint64]common.Address{
	BSCMainnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	BSCTestnet: common.HexToAddress("0x0000000000000000000000000000000000000000"),
}

// DAI - legacy, rebranded to USDS
var daiAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet: common.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f"),
	BSCMainnet:      common.HexToAddress("0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3"),
	BaseMainnet:     common.HexToAddress("0x50c5725949a6f0c72e6c4a641f24049a917db0cb"),
	ArbitrumMainnet: common.HexToAddress("0xda10009cbd5d07dd0cecc66161fc93d7c9000da1"),
	LineaMainnet:    common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
	EthereumHoodi:   common.HexToAddress("0xcafbbad55eb09efe7bec8408cff9932be7d9a7fa"),
	EthereumSepolia: common.HexToAddress("0x3e622317f8c93f7328350cf0b56d9ed4c620c5d6"),
	UnichainMainnet: common.HexToAddress("0x20cab320a855b39f724131c69424240519573f81"),
	UnichainSepolia: common.HexToAddress("0x35f965903a85e7528437c3ce0b4bdfbc4e5fc27c"),
}

// USDS - stable, rebranded from DAI
var usdsAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet: common.HexToAddress("0xdc035d45d973e3ec169d2276ddab16f1e407384f"),
	BaseMainnet:     common.HexToAddress("0x820c137fa70c8691f0e44dc420a5e53c168921dc"),
	ArbitrumMainnet: common.HexToAddress("0x6491c05a82219b8d1479057361ff1654749b876b"),
}

// USDC (EVM)
var usdcEVMAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
	OptimismMainnet: common.HexToAddress("0x0b2c639c533813f4aa9d7837caf62653d097ff85"),
	ArbitrumMainnet: common.HexToAddress("0xaf88d065e77c8cc2239327c5edb3a432268e5831"),
	BaseMainnet:     common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"),
	LineaMainnet:    common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
	EthereumHoodi:   common.HexToAddress("0x8c5476bfd428dce0f9b2315afb63b3976c6a2b50"),
	EthereumSepolia: common.HexToAddress("0x1c7d4b196cb0c7b01d743fbc6116a902379c7238"),
	OptimismSepolia: common.HexToAddress("0x5fd84259d66cd46123540766be93dfe6d43130d7"),
	ArbitrumSepolia: common.HexToAddress("0x75faf114eafb1bdbe2f0316df893fd58ce46aa4d"),
	BaseSepolia:     common.HexToAddress("0x036cbd53842c5426634e7929541ec2318f3dcf7e"),
	LineaSepolia:    common.HexToAddress("0xfece4462d57bd51a6a552365a011b95f0e16d9b7"),
	AbstractMainnet: common.HexToAddress("0x84a71ccd554cc1b02749b35d22F684cc8ec987e1"),
	AbstractTestnet: common.HexToAddress("0x572f4901f03055ffc1d936a60ccc3cbf13911be3"),
	UnichainMainnet: common.HexToAddress("0x078d782b760474a361dda0af3839290b0ef57ad6"),
	UnichainSepolia: common.HexToAddress("0x31d0220469e10c4e71834a79b1f276d740d3768f"),
	InkMainnet:      common.HexToAddress("0x2d270e6886d130d724215a266106e6832161eaed"),
	InkSepolia:      common.HexToAddress("0xfabab97dc620294d2b0b0e46c68964e326300ac4"),
	ZkSyncMainnet:   common.HexToAddress("0x1d17cbcf0d6d143135ae902365d2e5e2a16538d4"),
	ZkSyncSepolia:   common.HexToAddress("0xe045de5638162fa134807cb558e15a3f5a7f8535"),
	SoneiumMainnet:  common.HexToAddress("0xba9986d2381edf1da03b0b9c1f8b00dc4aacc369"),
	ScrollMainnet:   common.HexToAddress("0x06efdbff2a14a7c8e15944d1f4a48f9f95f663a4"),
}

// USDC (BSC)
var usdcBSCAddressesByChainID = map[uint64]common.Address{
	BSCMainnet: common.HexToAddress("0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"),
}

// SNT
var sntAddressesByChainID = map[uint64]common.Address{
	EthereumMainnet: common.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
	OptimismMainnet: common.HexToAddress("0x650af3c15af43dcb218406d30784416d64cfb6b2"),
	ArbitrumMainnet: common.HexToAddress("0x707f635951193ddafbb40971a0fcaab8a6415160"),
	BaseMainnet:     common.HexToAddress("0x662015ec830df08c0fc45896fab726542e8ac09e"),
	LineaMainnet:    common.HexToAddress("0xa3c26a308ac52520320ebcafdba0bb0aaa105ee8"),
	EthereumHoodi:   common.HexToAddress("0x0B5DAd18B8791ddb24252B433ec4f21f9e6e5Ed0"), // hoodi
	EthereumSepolia: common.HexToAddress("0xE452027cdEF746c7Cd3DB31CB700428b16cD8E51"),
	OptimismSepolia: common.HexToAddress("0x0b5dad18b8791ddb24252b433ec4f21f9e6e5ed0"),
	BaseSepolia:     common.HexToAddress("0xfdb3b57944943a7724fcc0520ee2b10659969a06"),
	LineaSepolia:    common.HexToAddress("0x4f3b44bdddb0e2f94a85d75294d0a38e211be6a8"),
}

func allMandatoryTokens() map[uint64][]common.Address {
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
	for chainID, address := range usdsAddressesByChainID {
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

	return allAddresses
}

func MandatoryTokens() []string {
	allAddresses := allMandatoryTokens()
	mandatoryTokens := make([]string, 0)
	for chainID, addresses := range allAddresses {
		for _, address := range addresses {
			mandatoryTokens = append(mandatoryTokens, types.TokenKey(chainID, address))
		}
	}
	return mandatoryTokens
}

func MandatoryTokensByChainID(chainID uint64) []string {
	allAddresses := allMandatoryTokens()
	mandatoryTokens := make([]string, 0)
	for cID, addresses := range allAddresses {
		if cID != chainID {
			continue
		}
		for _, address := range addresses {
			mandatoryTokens = append(mandatoryTokens, types.TokenKey(cID, address))
		}
	}
	return mandatoryTokens
}

func SkippedTokenKeys() []string {
	return []string{
		types.TokenKey(OptimismMainnet, common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")),
		types.TokenKey(OptimismSepolia, common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")),
		types.TokenKey(BSCMainnet, common.HexToAddress("0x683e9dcf085e5efcc7925858aace94d4b8882024")), // TANGYUAN
		types.TokenKey(BSCMainnet, common.HexToAddress("0x5ca42204cdaa70d5c773946e69de942b85ca6706")), // POSI
	}
}
