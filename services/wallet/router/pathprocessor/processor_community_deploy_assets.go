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
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type CommunityDeployAssetsProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunityDeployAssetsProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunityDeployAssetsProcessor {
	return &CommunityDeployAssetsProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunityDeployAssetsErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunityDeployAssetsName, err)
}

func (s *CommunityDeployAssetsProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunityDeployAssetsName
}

func (s *CommunityDeployAssetsProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunityDeployAssetsProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *CommunityDeployAssetsProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	assetsABI, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return []byte{}, err
	}

	data, err := assetsABI.Pack("" /*constructor name is empty*/, params.CommunityParams.DeploymentParameters.Name,
		params.CommunityParams.DeploymentParameters.Symbol, pathProcessorCommon.CommunityDeploymentTokenDecimals,
		params.CommunityParams.DeploymentParameters.GetSupply(), params.CommunityParams.DeploymentParameters.TokenURI,
		params.CommunityParams.DeploymentParameters.OwnerTokenAddress,
		params.CommunityParams.DeploymentParameters.MasterTokenAddress)
	if err != nil {
		return []byte{}, err
	}

	return append(common.FromHex(assets.AssetsBin), data...), nil
}

func (s *CommunityDeployAssetsProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createCommunityDeployAssetsErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunityDeployAssetsErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    nil,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createCommunityDeployAssetsErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunityDeployAssetsProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunityDeployAssetsProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunityDeployAssetsProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunityDeployAssetsProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *CommunityDeployAssetsProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *CommunityDeployAssetsProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return communitytokendeployer.ContractAddress(params.FromChain.ChainID)
}
