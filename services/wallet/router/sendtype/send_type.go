package sendtype

import (
	"math/big"

	"github.com/status-im/status-go/params"
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
		return pathProcessorName == pathProcessorCommon.ProcessorTransferName ||
			walletCommon.IsProcessorBridge(pathProcessorName)
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
		} else if s != ENSRelease {
			return false
		}
	}

	return true
}

func (s SendType) IsAvailableBetween(from, to *params.Network) bool {
	if s.IsCollectiblesTransfer() ||
		s.IsEnsTransfer() ||
		s.IsStickersTransfer() ||
		s.IsCommunityRelatedTransfer() ||
		s == Swap {
		return from.ChainID == to.ChainID
	}

	if s == Bridge {
		return from.ChainID != to.ChainID
	}

	return true
}

func (s SendType) IsAvailableFor(network *params.Network) bool {
	// Check for Swap specific networks
	if s == Swap {
		swapAllowedNetworks := map[uint64]bool{
			walletCommon.EthereumMainnet: true,
			walletCommon.OptimismMainnet: true,
			walletCommon.ArbitrumMainnet: true,
			walletCommon.BaseMainnet:     true,
			walletCommon.BSCMainnet:      true,
		}
		_, ok := swapAllowedNetworks[network.ChainID]
		return ok
	}

	if s == Bridge {
		bridgeAllowedNetworks := map[uint64]bool{
			walletCommon.EthereumMainnet: true,
			walletCommon.EthereumSepolia: true,
			walletCommon.OptimismMainnet: true,
			walletCommon.OptimismSepolia: true,
			walletCommon.ArbitrumMainnet: true,
			walletCommon.ArbitrumSepolia: true,
			walletCommon.BaseMainnet:     true,
			walletCommon.BaseSepolia:     true,
		}
		_, ok := bridgeAllowedNetworks[network.ChainID]
		return ok
	}

	if s.IsEnsTransfer() || s.IsStickersTransfer() {
		return network.ChainID == walletCommon.EthereumMainnet || network.ChainID == walletCommon.EthereumSepolia
	}

	return true
}
