package pathprocessor

const (
	IncreaseEstimatedGasFactor = 1.2
	SevenDaysInSeconds         = 60 * 60 * 24 * 7

	StatusDomain = ".stateofus.eth"
	EthDomain    = ".eth"

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

func IsProcessorBridge(name string) bool {
	return name == ProcessorBridgeHopName || name == ProcessorBridgeCelerName
}

func IsProcessorSwap(name string) bool {
	return name == ProcessorSwapParaswapName
}
