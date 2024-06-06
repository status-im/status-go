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

func (s *TransferProcessor) Name() string {
	return ProcessorTransferName
}

func (s *TransferProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == params.ToChain.ChainID && params.FromToken != nil && params.ToToken == nil, nil
}

func (s *TransferProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *TransferProcessor) PackTxInputData(params ProcessorInputParams, contractType string) ([]byte, error) {
	if params.FromToken.IsNative() {
		return []byte("eth_sendRawTransaction"), nil
	} else {
		abi, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
		if err != nil {
			return []byte{}, err
		}
		return abi.Pack("transfer",
			params.ToAddr,
			params.AmountIn,
		)
	}
}

func (s *TransferProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	estimation := uint64(0)
	var err error

	input, err := s.PackTxInputData(params, "")
	if err != nil {
		return 0, err
	}

	if params.FromToken.IsNative() {
		estimation, err = s.transactor.EstimateGas(params.FromChain, params.FromAddr, params.ToAddr, params.AmountIn, input)
		if err != nil {
			return 0, err
		}
	} else {
		ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
		if err != nil {
			return 0, err
		}

		ctx := context.Background()

		msg := ethereum.CallMsg{
			From: params.FromAddr,
			To:   &params.FromToken.Address,
			Data: input,
		}

		estimation, err = ethClient.EstimateGas(ctx, msg)
		if err != nil {
			return 0, err
		}

	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *TransferProcessor) BuildTx(params ProcessorInputParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	if params.FromToken.IsNative() {
		sendArgs := &MultipathProcessorTxArgs{
			TransferTx: &transactions.SendTxArgs{
				From:  types.Address(params.FromAddr),
				To:    &toAddr,
				Value: (*hexutil.Big)(params.AmountIn),
				Data:  types.HexBytes("0x0"),
			},
			ChainID: params.FromChain.ChainID,
		}

		return s.BuildTransaction(sendArgs)
	}
	abi, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
	if err != nil {
		return nil, err
	}
	input, err := abi.Pack("transfer",
		params.ToAddr,
		params.AmountIn,
	)
	if err != nil {
		return nil, err
	}
	sendArgs := &MultipathProcessorTxArgs{
		TransferTx: &transactions.SendTxArgs{
			From:  types.Address(params.FromAddr),
			To:    &toAddr,
			Value: (*hexutil.Big)(ZeroBigIntValue),
			Data:  input,
		},
		ChainID: params.FromChain.ChainID,
	}

	return s.BuildTransaction(sendArgs)
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
