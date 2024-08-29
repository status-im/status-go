package router

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
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

func (s SendType) FetchPrices(marketManager *market.Manager, tokenID string) (map[string]float64, error) {
	symbols := []string{tokenID, "ETH"}
	if s.IsCollectiblesTransfer() {
		symbols = []string{"ETH"}
	}

	pricesMap, err := marketManager.FetchPrices(symbols, []string{"USD"})
	if err != nil {
		return nil, err
	}
	prices := make(map[string]float64, 0)
	for symbol, pricePerCurrency := range pricesMap {
		prices[symbol] = pricePerCurrency["USD"]
	}
	if s.IsCollectiblesTransfer() {
		prices[tokenID] = 0
	}
	return prices, nil
}

func (s SendType) FindToken(tokenManager *token.Manager, collectibles *collectibles.Service, account common.Address, network *params.Network, tokenID string) *token.Token {
	if !s.IsCollectiblesTransfer() {
		return tokenManager.FindToken(network, tokenID)
	}

	parts := strings.Split(tokenID, ":")
	contractAddress := common.HexToAddress(parts[0])
	collectibleTokenID, success := new(big.Int).SetString(parts[1], 10)
	if !success {
		return nil
	}
	uniqueID, err := collectibles.GetOwnedCollectible(walletCommon.ChainID(network.ChainID), account, contractAddress, collectibleTokenID)
	if err != nil || uniqueID == nil {
		return nil
	}

	return &token.Token{
		Address:  contractAddress,
		Symbol:   collectibleTokenID.String(),
		Decimals: 0,
		ChainID:  network.ChainID,
	}
}

// TODO: remove this function once we fully move to routerV2
func (s SendType) isTransfer(routerV2Logic bool) bool {
	return s == Transfer ||
		s == Bridge && routerV2Logic ||
		s == Swap ||
		s.IsCollectiblesTransfer()
}

func (s SendType) needL1Fee() bool {
	return !s.IsEnsTransfer() && !s.IsStickersTransfer()
}

// canUseProcessor is used to check if certain SendType can be used with a given path processor
func (s SendType) canUseProcessor(p pathprocessor.PathProcessor) bool {
	pathProcessorName := p.Name()
	switch s {
	case Transfer:
		return pathProcessorName == pathprocessor.ProcessorTransferName ||
			pathprocessor.IsProcessorBridge(pathProcessorName)
	case Bridge:
		return pathprocessor.IsProcessorBridge(pathProcessorName)
	case Swap:
		return pathprocessor.IsProcessorSwap(pathProcessorName)
	case ERC721Transfer:
		return pathProcessorName == pathprocessor.ProcessorERC721Name
	case ERC1155Transfer:
		return pathProcessorName == pathprocessor.ProcessorERC1155Name
	case ENSRegister:
		return pathProcessorName == pathprocessor.ProcessorENSRegisterName
	case ENSRelease:
		return pathProcessorName == pathprocessor.ProcessorENSReleaseName
	case ENSSetPubKey:
		return pathProcessorName == pathprocessor.ProcessorENSPublicKeyName
	case StickersBuy:
		return pathProcessorName == pathprocessor.ProcessorStickersBuyName
	default:
		return true
	}
}

func (s SendType) processZeroAmountInProcessor(amountIn *big.Int, amountOut *big.Int, processorName string) bool {
	if amountIn.Cmp(pathprocessor.ZeroBigIntValue) == 0 {
		if s == Transfer {
			if processorName != pathprocessor.ProcessorTransferName {
				return false
			}
		} else if s == Swap {
			if amountOut.Cmp(pathprocessor.ZeroBigIntValue) == 0 {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func (s SendType) isAvailableBetween(from, to *params.Network) bool {
	if s.IsCollectiblesTransfer() ||
		s.IsEnsTransfer() ||
		s.IsStickersTransfer() ||
		s == Swap {
		return from.ChainID == to.ChainID
	}

	if s == Bridge {
		return from.ChainID != to.ChainID
	}

	return true
}

func (s SendType) isAvailableFor(network *params.Network) bool {
	// Set of network ChainIDs allowed for any type of transaction
	allAllowedNetworks := map[uint64]bool{
		walletCommon.EthereumMainnet: true,
		walletCommon.EthereumGoerli:  true,
		walletCommon.EthereumSepolia: true,
	}

	// Additional specific networks for the Swap SendType
	swapAllowedNetworks := map[uint64]bool{
		walletCommon.EthereumMainnet: true,
		walletCommon.OptimismMainnet: true,
		walletCommon.ArbitrumMainnet: true,
	}

	// Check for Swap specific networks
	if s == Swap {
		return swapAllowedNetworks[network.ChainID]
	}

	if s.IsEnsTransfer() || s.IsStickersTransfer() {
		return network.ChainID == walletCommon.EthereumMainnet || network.ChainID == walletCommon.EthereumSepolia
	}

	// Check for any SendType available for all networks
	if s == Transfer || s == Bridge || s.IsCollectiblesTransfer() || allAllowedNetworks[network.ChainID] {
		return true
	}

	return false
}

// TODO: remove this function once we fully move to routerV2
func (s SendType) EstimateGas(ensService *ens.Service, stickersService *stickers.Service, network *params.Network, from common.Address, tokenID string) uint64 {
	tx := transactions.SendTxArgs{
		From:  (types.Address)(from),
		Value: (*hexutil.Big)(pathprocessor.ZeroBigIntValue),
	}
	switch s {
	case ENSRegister:
		estimate, err := ensService.API().RegisterEstimate(context.Background(), network.ChainID, tx, EstimateUsername, EstimatePubKey)
		if err != nil {
			return 400000
		}
		return estimate

	case ENSRelease:
		estimate, err := ensService.API().ReleaseEstimate(context.Background(), network.ChainID, tx, EstimateUsername)
		if err != nil {
			return 200000
		}
		return estimate

	case ENSSetPubKey:
		estimate, err := ensService.API().SetPubKeyEstimate(context.Background(), network.ChainID, tx, fmt.Sprint(EstimateUsername, ".stateofus.eth"), EstimatePubKey)
		if err != nil {
			return 400000
		}
		return estimate

	case StickersBuy:
		packID := &bigint.BigInt{Int: big.NewInt(2)}
		estimate, err := stickersService.API().BuyEstimate(context.Background(), network.ChainID, (types.Address)(from), packID)
		if err != nil {
			return 400000
		}
		return estimate

	default:
		return 0
	}
}
