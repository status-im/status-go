package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/contracts/ierc20"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type TransferProcessor struct {
	ethClientGetter rpc.EthClientGetter
	transactor      transactions.TransactorIface
}

func NewTransferProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *TransferProcessor {
	return &TransferProcessor{ethClientGetter: ethClientGetter, transactor: transactor}
}

func createTransferErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorTransferName, err)
}

func (s *TransferProcessor) Name() string {
	return pathProcessorCommon.ProcessorTransferName
}

func (s *TransferProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromToken == nil || params.ToToken == nil {
		return false, ErrToAndFromTokensMustBeSet
	}
	if params.FromToken.ChainID != params.ToToken.ChainID ||
		params.FromToken.Address != params.ToToken.Address {
		return false, ErrFromAndToTokensMustBeSame
	}
	return true, nil
}

func (s *TransferProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *TransferProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	if params.TestsMode {
		return []byte{}, nil
	}
	if params.FromToken.IsNative() {
		return []byte{}, nil
	} else {
		abi, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
		if err != nil {
			return []byte{}, createTransferErrorResponse(err)
		}
		return abi.Pack("transfer",
			params.ToAddr,
			params.AmountIn,
		)
	}
}

func (s *TransferProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	estimation := uint64(0)
	var err error

	if params.FromToken.IsNative() {
		estimation, err = s.transactor.EstimateGas(params.FromChain.ChainID, params.FromAddr, params.ToAddr, params.AmountIn, input)
		if err != nil {
			return 0, createTransferErrorResponse(err)
		}
	} else {
		ethClient, err := s.ethClientGetter.EthClient(params.FromChain.ChainID)
		if err != nil {
			return 0, createTransferErrorResponse(err)
		}

		ctx := context.Background()

		msg := ethereum.CallMsg{
			From: params.FromAddr,
			To:   &params.FromToken.Address,
			Data: input,
		}

		estimation, err = ethClient.EstimateGas(ctx, msg)
		if err != nil {
			return 0, createTransferErrorResponse(err)
		}

	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *TransferProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *TransferProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *TransferProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return common.Address{}, nil
}
