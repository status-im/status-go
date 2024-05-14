package wallet

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/bridge"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
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

func (s SendType) FetchPrices(service *Service, tokenID string) (map[string]float64, error) {
	symbols := []string{tokenID, "ETH"}
	if s.IsCollectiblesTransfer() {
		symbols = []string{"ETH"}
	}

	pricesMap, err := service.marketManager.FetchPrices(symbols, []string{"USD"})
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

func (s SendType) FindToken(service *Service, account common.Address, network *params.Network, tokenID string) *token.Token {
	if !s.IsCollectiblesTransfer() {
		return service.tokenManager.FindToken(network, tokenID)
	}

	parts := strings.Split(tokenID, ":")
	contractAddress := common.HexToAddress(parts[0])
	collectibleTokenID, success := new(big.Int).SetString(parts[1], 10)
	if !success {
		return nil
	}
	uniqueID, err := service.collectibles.GetOwnedCollectible(walletCommon.ChainID(network.ChainID), account, contractAddress, collectibleTokenID)
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

func (s SendType) isTransfer() bool {
	return s == Transfer || s == Swap || s.IsCollectiblesTransfer()
}

func (s SendType) needL1Fee() bool {
	return s != ENSRegister && s != ENSRelease && s != ENSSetPubKey && s != StickersBuy
}

func (s SendType) isAvailableBetween(from, to *params.Network) bool {
	if s.IsCollectiblesTransfer() {
		return from.ChainID == to.ChainID
	}

	if s == Bridge {
		return from.ChainID != to.ChainID
	}

	if s == Swap {
		return from.ChainID == to.ChainID
	}

	return true
}

func (s SendType) canUseBridge(b bridge.Bridge) bool {
	if s == ERC721Transfer && b.Name() != ERC721TransferString {
		return false
	}

	if s != ERC721Transfer && b.Name() == ERC721TransferString {
		return false
	}

	if s == ERC1155Transfer && b.Name() != ERC1155TransferString {
		return false
	}

	if s != ERC1155Transfer && b.Name() == ERC1155TransferString {
		return false
	}

	return true
}

func (s SendType) isAvailableFor(network *params.Network) bool {
	if s == Swap {
		return network.ChainID == walletCommon.EthereumMainnet ||
			network.ChainID == walletCommon.OptimismMainnet ||
			network.ChainID == walletCommon.ArbitrumMainnet
	}

	if s == Transfer || s == Bridge || s.IsCollectiblesTransfer() {
		return true
	}

	if network.ChainID == walletCommon.EthereumMainnet ||
		network.ChainID == walletCommon.EthereumGoerli ||
		network.ChainID == walletCommon.EthereumSepolia {
		return true
	}

	return false
}

func (s SendType) EstimateGas(service *Service, network *params.Network, from common.Address, tokenID string) uint64 {
	tx := transactions.SendTxArgs{
		From:  (types.Address)(from),
		Value: (*hexutil.Big)(zero),
	}
	if s == ENSRegister {
		estimate, err := service.ens.API().RegisterEstimate(context.Background(), network.ChainID, tx, EstimateUsername, EstimatePubKey)
		if err != nil {
			return 400000
		}
		return estimate
	}

	if s == ENSRelease {
		estimate, err := service.ens.API().ReleaseEstimate(context.Background(), network.ChainID, tx, EstimateUsername)
		if err != nil {
			return 200000
		}
		return estimate
	}

	if s == ENSSetPubKey {
		estimate, err := service.ens.API().SetPubKeyEstimate(context.Background(), network.ChainID, tx, fmt.Sprint(EstimateUsername, ".stateofus.eth"), EstimatePubKey)
		if err != nil {
			return 400000
		}
		return estimate
	}

	if s == StickersBuy {
		packID := &bigint.BigInt{Int: big.NewInt(2)}
		estimate, err := service.stickers.API().BuyEstimate(context.Background(), network.ChainID, (types.Address)(from), packID)
		if err != nil {
			return 400000
		}
		return estimate
	}

	return 0
}
