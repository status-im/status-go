package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/contracts/ierc1155"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type ERC1155TxArgs struct {
	wallettypes.SendTxArgs
	TokenID   *hexutil.Big   `json:"tokenId"`
	Recipient common.Address `json:"recipient"`
	Amount    *hexutil.Big   `json:"amount"`
}

type ERC1155Processor struct {
	ethClientGetter rpc.EthClientGetter
	transactor      transactions.TransactorIface
}

func NewERC1155Processor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *ERC1155Processor {
	return &ERC1155Processor{ethClientGetter: ethClientGetter, transactor: transactor}
}

func createERC1155ErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorERC1155Name, err)
}

func (s *ERC1155Processor) Name() string {
	return pathProcessorCommon.ProcessorERC1155Name
}

func (s *ERC1155Processor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == params.ToChain.ChainID, nil
}

func (s *ERC1155Processor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *ERC1155Processor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	if params.FromToken == nil || params.FromToken.CollectibleTokenID == nil {
		return nil, ErrNoTokenSet
	}

	abi, err := abi.JSON(strings.NewReader(ierc1155.Ierc1155ABI))
	if err != nil {
		return []byte{}, createERC1155ErrorResponse(err)
	}

	return abi.Pack("safeTransferFrom",
		params.FromAddr,
		params.ToAddr,
		params.FromToken.CollectibleTokenID.ToInt(),
		params.AmountIn,
		[]byte{},
	)
}

func (s *ERC1155Processor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	ethClient, err := s.ethClientGetter.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createERC1155ErrorResponse(err)
	}

	value := new(big.Int)

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createERC1155ErrorResponse(err)
	}
	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *ERC1155Processor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *ERC1155Processor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ERC1155Processor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.FromToken.Address, nil
}
