package bridge

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type ENSRegisterBridge struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
	ensService    *ens.Service
}

func NewENSRegisterBridge(rpcClient *rpc.Client, transactor transactions.TransactorIface, ensService *ens.Service) *ENSRegisterBridge {
	return &ENSRegisterBridge{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		// rpcClient:  rpcClient,
		transactor: transactor,
		ensService: ensService,
	}
}

func (s *ENSRegisterBridge) Name() string {
	return ENSRegisterName
}

func (s *ENSRegisterBridge) GetPriceForRegisteringEnsName(chainID uint64) (*big.Int, error) {
	registryAddr, err := s.ensService.API().GetRegistrarAddress(context.Background(), chainID)
	if err != nil {
		return nil, err
	}
	registrar, err := s.contractMaker.NewUsernameRegistrar(chainID, registryAddr)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
	return registrar.GetPrice(callOpts)
}

func (s *ENSRegisterBridge) AvailableFor(params BridgeParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSRegisterBridge) CalculateFees(params BridgeParams) (*big.Int, *big.Int, error) {
	return big.NewInt(0), big.NewInt(0), nil
}

func (s *ENSRegisterBridge) PackTxInputData(params BridgeParams, contractType string) ([]byte, error) {
	price, err := s.GetPriceForRegisteringEnsName(params.FromChain.ChainID)
	if err != nil {
		return []byte{}, err
	}

	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return []byte{}, err
	}

	x, y := extractCoordinates(params.PublicKey)
	extraData, err := registrarABI.Pack("register", usernameToLabel(params.Username), params.FromAddr, x, y)
	if err != nil {
		return []byte{}, err
	}

	sntABI, err := abi.JSON(strings.NewReader(snt.SNTABI))
	if err != nil {
		return []byte{}, err
	}

	registryAddr, err := s.ensService.API().GetRegistrarAddress(context.Background(), params.FromChain.ChainID)
	if err != nil {
		return []byte{}, err
	}

	return sntABI.Pack("approveAndCall", registryAddr, price, extraData)
}

func (s *ENSRegisterBridge) EstimateGas(params BridgeParams) (uint64, error) {
	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, err
	}

	input, err := s.PackTxInputData(params, "")
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
		Value: big.NewInt(0),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, err
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *ENSRegisterBridge) BuildTx(params BridgeParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	inputData, err := s.PackTxInputData(params, "")
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

func (s *ENSRegisterBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, verifiedAccount)
}

func (s *ENSRegisterBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx)
}

func (s *ENSRegisterBridge) CalculateAmountOut(params BridgeParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ENSRegisterBridge) GetContractAddress(params BridgeParams) (common.Address, error) {
	return snt.ContractAddress(params.FromChain.ChainID)
}
