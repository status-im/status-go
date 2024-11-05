package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/ensresolver"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type ENSRegisterProcessor struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
	ensResolver   *ensresolver.EnsResolver
}

func NewENSRegisterProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, ensResolver *ensresolver.EnsResolver) *ENSRegisterProcessor {
	return &ENSRegisterProcessor{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		transactor:  transactor,
		ensResolver: ensResolver,
	}
}

func createENSRegisterProcessorErrorResponse(err error) error {
	return createErrorResponse(walletCommon.ProcessorENSRegisterName, err)
}

func (s *ENSRegisterProcessor) Name() string {
	return walletCommon.ProcessorENSRegisterName
}

func (s *ENSRegisterProcessor) GetPriceForRegisteringEnsName(chainID uint64) (*big.Int, error) {
	registryAddr, err := s.ensResolver.GetRegistrarAddress(context.Background(), chainID)
	if err != nil {
		return nil, createENSRegisterProcessorErrorResponse(err)
	}
	registrar, err := s.contractMaker.NewUsernameRegistrar(chainID, registryAddr)
	if err != nil {
		return nil, createENSRegisterProcessorErrorResponse(err)
	}

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
	return registrar.GetPrice(callOpts)
}

func (s *ENSRegisterProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSRegisterProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *ENSRegisterProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	price, err := s.GetPriceForRegisteringEnsName(params.FromChain.ChainID)
	if err != nil {
		return []byte{}, createENSRegisterProcessorErrorResponse(err)
	}

	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return []byte{}, createENSRegisterProcessorErrorResponse(err)
	}

	x, y := walletCommon.ExtractCoordinates(params.PublicKey)
	extraData, err := registrarABI.Pack("register", walletCommon.UsernameToLabel(params.Username), params.FromAddr, x, y)
	if err != nil {
		return []byte{}, createENSRegisterProcessorErrorResponse(err)
	}

	sntABI, err := abi.JSON(strings.NewReader(snt.SNTABI))
	if err != nil {
		return []byte{}, createENSRegisterProcessorErrorResponse(err)
	}

	registryAddr, err := s.ensResolver.GetRegistrarAddress(context.Background(), params.FromChain.ChainID)
	if err != nil {
		return []byte{}, createENSRegisterProcessorErrorResponse(err)
	}

	return sntABI.Pack("approveAndCall", registryAddr, price, extraData)
}

func (s *ENSRegisterProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *ENSRegisterProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *ENSRegisterProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *ENSRegisterProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *ENSRegisterProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ENSRegisterProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return snt.ContractAddress(params.FromChain.ChainID)
}
