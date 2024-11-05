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
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/ensresolver"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/transactions"
)

type ENSReleaseProcessor struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
	ensResolver   *ensresolver.EnsResolver
}

func NewENSReleaseProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, ensResolver *ensresolver.EnsResolver) *ENSReleaseProcessor {
	return &ENSReleaseProcessor{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		transactor:  transactor,
		ensResolver: ensResolver,
	}
}

func createENSReleaseErrorResponse(err error) error {
	return createErrorResponse(walletCommon.ProcessorENSReleaseName, err)
}

func (s *ENSReleaseProcessor) Name() string {
	return walletCommon.ProcessorENSReleaseName
}

func (s *ENSReleaseProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSReleaseProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *ENSReleaseProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return []byte{}, createENSReleaseErrorResponse(err)
	}

	name := getNameFromEnsUsername(params.Username)
	return registrarABI.Pack("release", walletCommon.UsernameToLabel(name))
}

func (s *ENSReleaseProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
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
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createENSReleaseErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *ENSReleaseProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *ENSReleaseProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *ENSReleaseProcessor) BuildTransactionV2(sendArgs *transactions.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *ENSReleaseProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ENSReleaseProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	addr, err := s.ensResolver.GetRegistrarAddress(context.Background(), params.FromChain.ChainID)
	if err != nil {
		return common.Address{}, err
	}
	if addr == walletCommon.ZeroAddress() {
		return common.Address{}, ErrENSRegistrarNotFound
	}
	return addr, nil
}
