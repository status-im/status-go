package sendtype

import (
	"math/big"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
)

type SendType int

const (
	Transfer SendType = iota
	ENSRegister
	ENSRelease
	ENSSetPubKey
	StickersBuy
	Bridge
	ERC721Transfer
	ERC1155Transfer
	Swap
	CommunityBurn
	CommunityDeployAssets
	CommunityDeployCollectibles
	CommunityDeployOwnerToken
	CommunityMintTokens
	CommunityRemoteBurn
	CommunitySetSignerPubKey
)

func (s SendType) IsCollectiblesTransfer() bool {
	return s == ERC721Transfer || s == ERC1155Transfer
}

func (s SendType) IsEnsTransfer() bool {
	return s == ENSRegister || s == ENSRelease || s == ENSSetPubKey
}

func (s SendType) IsStickersTransfer() bool {
	return s == StickersBuy
}

func (s SendType) IsCommunityRelatedTransfer() bool {
	return s == CommunityDeployOwnerToken || s == CommunityDeployCollectibles || s == CommunityDeployAssets ||
		s == CommunityMintTokens || s == CommunityRemoteBurn || s == CommunityBurn || s == CommunitySetSignerPubKey
}

// canUseProcessor is used to check if certain SendType can be used with a given path processor
func (s SendType) CanUseProcessor(pathProcessorName string) bool {
	switch s {
	case Transfer:
		return pathProcessorName == pathProcessorCommon.ProcessorTransferName
	case Bridge:
		return walletCommon.IsProcessorBridge(pathProcessorName)
	case Swap:
		return walletCommon.IsProcessorSwap(pathProcessorName)
	case ERC721Transfer:
		return pathProcessorName == pathProcessorCommon.ProcessorERC721Name
	case ERC1155Transfer:
		return pathProcessorName == pathProcessorCommon.ProcessorERC1155Name
	case ENSRegister:
		return pathProcessorName == pathProcessorCommon.ProcessorENSRegisterName
	case ENSRelease:
		return pathProcessorName == pathProcessorCommon.ProcessorENSReleaseName
	case ENSSetPubKey:
		return pathProcessorName == pathProcessorCommon.ProcessorENSPublicKeyName
	case StickersBuy:
		return pathProcessorName == pathProcessorCommon.ProcessorStickersBuyName
	case CommunityBurn:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunityBurnName
	case CommunityDeployAssets:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunityDeployAssetsName
	case CommunityDeployCollectibles:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunityDeployCollectiblesName
	case CommunityDeployOwnerToken:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName
	case CommunityMintTokens:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunityMintTokensName
	case CommunityRemoteBurn:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunityRemoteBurnName
	case CommunitySetSignerPubKey:
		return pathProcessorName == pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName
	default:
		return true
	}
}

func (s SendType) ProcessZeroAmountInProcessor(amountIn *big.Int, amountOut *big.Int, processorName string) bool {
	if amountIn.Cmp(walletCommon.ZeroBigIntValue()) == 0 {
		if s == Transfer {
			if processorName != pathProcessorCommon.ProcessorTransferName {
				return false
			}
		} else if s == Swap {
			if amountOut.Cmp(walletCommon.ZeroBigIntValue()) == 0 {
				return false
			}
		} else if s.IsCommunityRelatedTransfer() {
			return true
		} else if s == ENSSetPubKey {
			return true
		} else if s != ENSRelease {
			return false
		}
	}

	return true
}

func (s SendType) IsAvailableBetween(fromChainID, toChainID uint64) bool {
	if s.IsCollectiblesTransfer() ||
		s.IsEnsTransfer() ||
		s.IsStickersTransfer() ||
		s.IsCommunityRelatedTransfer() ||
		s == Swap {
		return fromChainID == toChainID
	}

	if s == Bridge {
		return fromChainID != toChainID
	}

	return true
}

func (s SendType) IsAvailableFor(chainID uint64) bool {
	// Check for Swap specific networks
	if s == Swap {
		swapAllowedNetworks := map[uint64]bool{
			walletCommon.EthereumMainnet: true,
			walletCommon.OptimismMainnet: true,
			walletCommon.ArbitrumMainnet: true,
			walletCommon.BaseMainnet:     true,
			walletCommon.BSCMainnet:      true,
			walletCommon.LineaMainnet:    true,
		}
		_, ok := swapAllowedNetworks[chainID]
		return ok
	}

	// Return true for Bridge as a real check is performed when AvailableFor for path processor is called
	if s == Bridge {
		return true
	}

	if s.IsEnsTransfer() || s.IsStickersTransfer() {
		return chainID == walletCommon.EthereumMainnet || chainID == walletCommon.EthereumSepolia || chainID == walletCommon.AnvilMainnet
	}

	return true
}
