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
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type ENSReleaseProcessor struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
	ensService    *ens.Service
}

func NewENSReleaseProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, ensService *ens.Service) *ENSReleaseProcessor {
	return &ENSReleaseProcessor{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
		ensService: ensService,
	}
}

func createENSReleaseErrorResponse(err error) error {
	return createErrorResponse(ProcessorENSReleaseName, err)
}

func (s *ENSReleaseProcessor) Name() string {
	return ProcessorENSReleaseName
}

func (s *ENSReleaseProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSReleaseProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return ZeroBigIntValue, ZeroBigIntValue, nil
}

func (s *ENSReleaseProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return []byte{}, createENSReleaseErrorResponse(err)
	}

	return registrarABI.Pack("release", ens.UsernameToLabel(params.Username))
}

func (s *ENSReleaseProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val, nil
			}
		}
		return 0, ErrNoEstimationFound
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createENSReleaseErrorResponse(err)
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createENSReleaseErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createENSReleaseErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: ZeroBigIntValue,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createENSReleaseErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *ENSReleaseProcessor) BuildTx(params ProcessorInputParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	inputData, err := s.PackTxInputData(params)
	if err != nil {
		return nil, createENSReleaseErrorResponse(err)
	}

	sendArgs := &MultipathProcessorTxArgs{
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

func (s *ENSReleaseProcessor) Send(sendArgs *MultipathProcessorTxArgs, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, verifiedAccount)
}

func (s *ENSReleaseProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs) (*ethTypes.Transaction, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx)
}

func (s *ENSReleaseProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ENSReleaseProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	addr, err := s.ensService.API().GetRegistrarAddress(context.Background(), params.FromChain.ChainID)
	if err != nil {
		return common.Address{}, err
	}
	if addr == ZeroAddress {
		return common.Address{}, ErrENSRegistrarNotFound
	}
	return addr, nil
}
