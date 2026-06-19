package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/services/ens/ensresolver"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

// setPubkeyABI is the single public-resolver method used to attach a chat key to
// an ENS name: `setPubkey(bytes32 node, bytes32 x, bytes32 y)`.
const setPubkeyABI = `[{"name":"setPubkey","inputs":[{"type":"bytes32"},{"type":"bytes32"},{"type":"bytes32"}],"outputs":[],"stateMutability":"nonpayable","type":"function"}]`

type ENSPublicKeyProcessor struct {
	contractMaker   *contracts.ContractMaker
	ethClientGetter rpc.EthClientGetter
	transactor      transactions.TransactorIface
	ensResolver     *ensresolver.EnsResolver
}

func NewENSPublicKeyProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface, ensResolver *ensresolver.EnsResolver) *ENSPublicKeyProcessor {
	return &ENSPublicKeyProcessor{
		contractMaker:   contracts.NewContractMaker(ethClientGetter),
		ethClientGetter: ethClientGetter,
		transactor:      transactor,
		ensResolver:     ensResolver,
	}
}

func createENSPublicKeyErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorENSPublicKeyName, err)
}

func (s *ENSPublicKeyProcessor) Name() string {
	return pathProcessorCommon.ProcessorENSPublicKeyName
}

func (s *ENSPublicKeyProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return params.FromChain.ChainID == walletCommon.EthereumMainnet || params.FromChain.ChainID == walletCommon.EthereumSepolia, nil
}

func (s *ENSPublicKeyProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *ENSPublicKeyProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	resolverABI, err := abi.JSON(strings.NewReader(setPubkeyABI))
	if err != nil {
		return []byte{}, createENSPublicKeyErrorResponse(err)
	}

	x, y := walletCommon.ExtractCoordinates(params.PublicKey)
	return resolverABI.Pack("setPubkey", walletCommon.NameHash(params.Username), x, y)
}

func (s *ENSPublicKeyProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
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

	ethClient, err := s.ethClientGetter.EthClient(params.FromChain.ChainID)
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

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
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
