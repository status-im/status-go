package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	communitytokens "github.com/status-im/status-go/contracts/community-tokens"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type CommunityDeployCollectiblesProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunityDeployCollectiblesProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunityDeployCollectiblesProcessor {
	return &CommunityDeployCollectiblesProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunityDeployCollectiblesErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunityDeployCollectiblesName, err)
}

func (s *CommunityDeployCollectiblesProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunityDeployCollectiblesName
}

func (s *CommunityDeployCollectiblesProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunityDeployCollectiblesProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *CommunityDeployCollectiblesProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
	if err != nil {
		return []byte{}, err
	}

	data, err := collectiblesABI.Pack("" /*constructor name is empty*/, params.CommunityParams.DeploymentParameters.Name,
		params.CommunityParams.DeploymentParameters.Symbol, params.CommunityParams.DeploymentParameters.GetSupply(),
		params.CommunityParams.DeploymentParameters.RemoteSelfDestruct, params.CommunityParams.DeploymentParameters.Transferable,
		params.CommunityParams.DeploymentParameters.TokenURI, params.CommunityParams.DeploymentParameters.OwnerTokenAddress,
		params.CommunityParams.DeploymentParameters.MasterTokenAddress)
	if err != nil {
		return []byte{}, err
	}

	return append(common.FromHex(collectibles.CollectiblesBin), data...), nil
}

func (s *CommunityDeployCollectiblesProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createCommunityDeployCollectiblesErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunityDeployCollectiblesErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    nil,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createCommunityDeployCollectiblesErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunityDeployCollectiblesProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunityDeployCollectiblesProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunityDeployCollectiblesProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunityDeployCollectiblesProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *CommunityDeployCollectiblesProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *CommunityDeployCollectiblesProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return communitytokendeployer.ContractAddress(params.FromChain.ChainID)
}
