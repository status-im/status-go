package bridge

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

type TransferBridge struct {
	rpcClient  *rpc.Client
	transactor transactions.TransactorIface
}

func NewTransferBridge(rpcClient *rpc.Client, transactor transactions.TransactorIface) *TransferBridge {
	return &TransferBridge{rpcClient: rpcClient, transactor: transactor}
}

func (s *TransferBridge) Name() string {
	return TransferName
}

func (s *TransferBridge) AvailableFor(params BridgeParams) (bool, error) {
	return params.FromChain.ChainID == params.ToChain.ChainID && params.FromToken != nil && params.ToToken == nil, nil
}

func (s *TransferBridge) CalculateFees(params BridgeParams) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *TransferBridge) PackTxInputData(params BridgeParams) ([]byte, error) {
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

func (s *TransferBridge) EstimateGas(params BridgeParams) (uint64, error) {
	estimation := uint64(0)
	var err error

	input, err := s.PackTxInputData(params)
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

func (s *TransferBridge) BuildTx(params BridgeParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	if params.FromToken.IsNative() {
		sendArgs := &TransactionBridge{
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
	sendArgs := &TransactionBridge{
		TransferTx: &transactions.SendTxArgs{
			From:  types.Address(params.FromAddr),
			To:    &toAddr,
			Value: (*hexutil.Big)(big.NewInt(0)),
			Data:  input,
		},
		ChainID: params.FromChain.ChainID,
	}

	return s.BuildTransaction(sendArgs)
}

func (s *TransferBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, verifiedAccount)
}

func (s *TransferBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx)
}

func (s *TransferBridge) CalculateAmountOut(params BridgeParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *TransferBridge) GetContractAddress(params BridgeParams) (common.Address, error) {
	return common.Address{}, nil
}
