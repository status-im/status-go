package common

const (
	IncreaseEstimatedGasFactor          = 1.05
	IncreaseEstimatedGasFactorForBridge = 1.2
	SevenDaysInSeconds                  = 60 * 60 * 24 * 7

	CommunityDeploymentTokenDecimals = uint8(18)

	ProcessorTransferName                    = "Transfer"
	ProcessorBridgeHopName                   = "Hop"
	ProcessorBridgeCelerName                 = "CBridge"
	ProcessorSwapParaswapName                = "Paraswap"
	ProcessorERC721Name                      = "ERC721Transfer"
	ProcessorERC1155Name                     = "ERC1155Transfer"
	ProcessorENSRegisterName                 = "ENSRegister"
	ProcessorENSReleaseName                  = "ENSRelease"
	ProcessorENSPublicKeyName                = "ENSPublicKey"
	ProcessorStickersBuyName                 = "StickersBuy"
	ProcessorCommunityDeployCollectiblesName = "CommunityDeployCollectibles"
	ProcessorCommunityDeployOwnerTokenName   = "CommunityDeployOwnerToken"
	ProcessorCommunityBurnName               = "CommunityBurn"
	ProcessorCommunityDeployAssetsName       = "CommunityDeployAssets"
	ProcessorCommunityMintTokensName         = "CommunityMintTokens"
	ProcessorCommunityRemoteBurnName         = "CommunityRemoteBurn"
	ProcessorCommunitySetSignerPubKeyName    = "CommunitySetSignerPubKey"
)
