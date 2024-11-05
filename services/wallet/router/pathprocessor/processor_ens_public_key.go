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
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/ensresolver"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type ENSPublicKeyProcessor struct {
	contractMaker *contracts.ContractMaker
	transactor    transactions.TransactorIface
	ensResolver   *ensresolver.EnsResolver
}

func NewENSPublicKeyProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, ensResolver *ensresolver.EnsResolver) *ENSPublicKeyProcessor {
	return &ENSPublicKeyProcessor{
		contractMaker: &contracts.ContractMaker{
			RPCClient: rpcClient,
		},
		transactor:  transactor,
		ensResolver: ensResolver,
	}
}

func createENSPublicKeyErrorResponse(err error) error {
	return createErrorResponse(walletCommon.ProcessorENSPublicKeyName, err)
}

func (s *ENSPublicKeyProcessor) Name() string {
	return walletCommon.ProcessorENSPublicKeyName
}

func (s *ENSPublicKeyProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSPublicKeyProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *ENSPublicKeyProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	resolverABI, err := abi.JSON(strings.NewReader(resolver.PublicResolverABI))
	if err != nil {
		return []byte{}, createENSPublicKeyErrorResponse(err)
	}

	x, y := walletCommon.ExtractCoordinates(params.PublicKey)
	return resolverABI.Pack("setPubkey", walletCommon.NameHash(params.Username), x, y)
}

func (s *ENSPublicKeyProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
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
		return 0, createENSPublicKeyErrorResponse(err)
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createENSPublicKeyErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createENSPublicKeyErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

func (s *ENSPublicKeyProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *ENSPublicKeyProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *ENSPublicKeyProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *ENSPublicKeyProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *ENSPublicKeyProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	addr, err := s.ensResolver.Resolver(context.Background(), params.FromChain.ChainID, params.Username)
	if err != nil {
		return common.Address{}, createENSPublicKeyErrorResponse(err)
	}
	if *addr == walletCommon.ZeroAddress() {
		return common.Address{}, ErrENSResolverNotFound
	}
	return *addr, nil
}
