package pathprocessor

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

//go:generate go tool mockgen -source=nft_handler.go -destination=nft_handler_mock_test.go -package=pathprocessor NFTHandler

// NFTHandler handling different types of NFT transfers
type NFTHandler interface {
	Name() string

	CanHandle(contractID thirdparty.ContractID) bool

	PackTxInputData(params ProcessorInputParams) ([]byte, error)

	EstimateGas(params ProcessorInputParams, input []byte, handlerName string) (uint64, error)

	BuildTransactionV2(
		transactor transactions.TransactorIface,
		sendArgs *wallettypes.SendTxArgs,
		lastUsedNonce int64,
	) (*ethTypes.Transaction, uint64, error)

	GetContractAddress(params ProcessorInputParams) (common.Address, error)
}

type BaseNFTHandler struct {
	ethClientGetter rpc.EthClientGetter
	transactor      transactions.TransactorIface
}

func NewBaseNFTHandler(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *BaseNFTHandler {
	return &BaseNFTHandler{
		ethClientGetter: ethClientGetter,
		transactor:      transactor,
	}
}

func (h *BaseNFTHandler) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (h *BaseNFTHandler) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (h *BaseNFTHandler) EstimateGas(params ProcessorInputParams, input []byte, handlerName string) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[handlerName]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	ethClient, err := h.ethClientGetter.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	estimation, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: new(big.Int),
		Data:  input,
	})
	if err != nil {
		return 0, err
	}

	return uint64(float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor), nil
}
