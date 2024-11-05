package common

import (
	"math/big"
	"strconv"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

type MultiTransactionIDType int64

const (
	NoMultiTransactionID = MultiTransactionIDType(0)
	HexAddressLength     = 42

	StatusDomain = "stateofus.eth"
	EthDomain    = "eth"

	EthSymbol  = "ETH"
	SntSymbol  = "SNT"
	SttSymbol  = "STT"
	UsdcSymbol = "USDC"
	HopSymbol  = "HOP"

	ProcessorTransferName     = "Transfer"
	ProcessorBridgeHopName    = "Hop"
	ProcessorBridgeCelerName  = "CBridge"
	ProcessorSwapParaswapName = "Paraswap"
	ProcessorERC721Name       = "ERC721Transfer"
	ProcessorERC1155Name      = "ERC1155Transfer"
	ProcessorENSRegisterName  = "ENSRegister"
	ProcessorENSReleaseName   = "ENSRelease"
	ProcessorENSPublicKeyName = "ENSPublicKey"
	ProcessorStickersBuyName  = "StickersBuy"
)

type ChainID uint64

const (
	UnknownChainID     uint64 = 0
	EthereumMainnet    uint64 = 1
	EthereumSepolia    uint64 = 11155111
	OptimismMainnet    uint64 = 10
	OptimismSepolia    uint64 = 11155420
	ArbitrumMainnet    uint64 = 42161
	ArbitrumSepolia    uint64 = 421614
	BinanceChainID     uint64 = 56 // obsolete?
	BinanceTestChainID uint64 = 97 // obsolete?
	AnvilMainnet       uint64 = 31337
)

var (
	SupportedNetworks = map[uint64]bool{
		EthereumMainnet: true,
		OptimismMainnet: true,
		ArbitrumMainnet: true,
	}

	SupportedTestNetworks = map[uint64]bool{
		EthereumSepolia: true,
		OptimismSepolia: true,
		ArbitrumSepolia: true,
	}
)

type ContractType byte

const (
	ContractTypeUnknown ContractType = iota
	ContractTypeERC20
	ContractTypeERC721
	ContractTypeERC1155
)

func ZeroAddress() ethCommon.Address {
	return ethCommon.Address{}
}

func ZeroBigIntValue() *big.Int {
	return big.NewInt(0)
}

func ZeroHash() ethCommon.Hash {
	return ethCommon.Hash{}
}

func (c ChainID) String() string {
	return strconv.FormatUint(uint64(c), 10)
}

func (c ChainID) ToUint() uint64 {
	return uint64(c)
}

func (c ChainID) IsMainnet() bool {
	switch uint64(c) {
	case EthereumMainnet, OptimismMainnet, ArbitrumMainnet:
		return true
	case EthereumSepolia, OptimismSepolia, ArbitrumSepolia:
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
	}
}

var AverageBlockDurationForChain = map[ChainID]time.Duration{
	ChainID(UnknownChainID):  time.Duration(12000) * time.Millisecond,
	ChainID(EthereumMainnet): time.Duration(12000) * time.Millisecond,
	ChainID(OptimismMainnet): time.Duration(400) * time.Millisecond,
	ChainID(ArbitrumMainnet): time.Duration(300) * time.Millisecond,
}
