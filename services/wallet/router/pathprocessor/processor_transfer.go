package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/transactions"
)

type TransferProcessor struct {
	rpcClient  *rpc.Client
	transactor transactions.TransactorIface
}

func NewTransferProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *TransferProcessor {
	return &TransferProcessor{rpcClient: rpcClient, transactor: transactor}
}

func createTransferErrorResponse(err error) error {
	return createErrorResponse(ProcessorTransferName, err)
}

func (s *TransferProcessor) Name() string {
	return ProcessorTransferName
}

func (s *TransferProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromChain == nil || params.ToChain == nil {
		return false, ErrNoChainSet
	}
	if params.FromToken == nil {
		return false, ErrNoTokenSet
	}
	if params.ToToken != nil {
		return false, ErrToTokenShouldNotBeSet
	}
	return params.FromChain.ChainID == params.ToChain.ChainID, nil
}

func (s *TransferProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *TransferProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	if params.FromToken.IsNative() {
		return []byte("eth_sendRawTransaction"), nil
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

func (s *TransferProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val, nil
			}
		}
		return 0, ErrNoEstimationFound
	}

	estimation := uint64(0)
	var err error

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createTransferErrorResponse(err)
	}

	if params.FromToken.IsNative() {
		estimation, err = s.transactor.EstimateGas(params.FromChain, params.FromAddr, params.ToAddr, params.AmountIn, input)
		if err != nil {
			return 0, createTransferErrorResponse(err)
		}
	} else {
		ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
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

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *TransferProcessor) Send(sendArgs *MultipathProcessorTxArgs, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, verifiedAccount)
}

func (s *TransferProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs) (*ethTypes.Transaction, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx)
}

func (s *TransferProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *TransferProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return common.Address{}, nil
}
