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
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type ENSReleaseBridge struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
	ensService    *ens.Service
}

func NewENSReleaseBridge(rpcClient *rpc.Client, transactor transactions.TransactorIface, ensService *ens.Service) *ENSReleaseBridge {
	return &ENSReleaseBridge{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
		ensService: ensService,
	}
}

func (s *ENSReleaseBridge) Name() string {
	return ENSReleaseName
}

func (s *ENSReleaseBridge) AvailableFor(params BridgeParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSReleaseBridge) CalculateFees(params BridgeParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *ENSReleaseBridge) PackTxInputData(params BridgeParams) ([]byte, error) {
	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return []byte{}, err
	}

	return registrarABI.Pack("release", usernameToLabel(params.Username))
}

func (s *ENSReleaseBridge) EstimateGas(params BridgeParams) (uint64, error) {
	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, err
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, err
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: ZeroBigIntValue,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, err
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *ENSReleaseBridge) BuildTx(params BridgeParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	inputData, err := s.PackTxInputData(params)
	if err != nil {
		return nil, err
	}

	sendArgs := &TransactionBridge{
		TransferTx: &transactions.SendTxArgs{
			From:  types.Address(params.FromAddr),
			To:    &toAddr,
			Value: (*hexutil.Big)(ZeroBigIntValue),
			Data:  inputData,
		},
		ChainID: params.FromChain.ChainID,
	}

	return s.BuildTransaction(sendArgs)
}

func (s *ENSReleaseBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, verifiedAccount)
}

func (s *ENSReleaseBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx)
}

func (s *ENSReleaseBridge) CalculateAmountOut(params BridgeParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ENSReleaseBridge) GetContractAddress(params BridgeParams) (common.Address, error) {
	return s.ensService.API().GetRegistrarAddress(context.Background(), params.FromChain.ChainID)
}
