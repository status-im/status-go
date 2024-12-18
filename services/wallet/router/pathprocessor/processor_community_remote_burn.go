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
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type CommunityRemoteBurnProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunityRemoteBurnProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunityRemoteBurnProcessor {
	return &CommunityRemoteBurnProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunityRemoteBurnErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunityRemoteBurnName, err)
}

func (s *CommunityRemoteBurnProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunityRemoteBurnName
}

func (s *CommunityRemoteBurnProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunityRemoteBurnProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *CommunityRemoteBurnProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
	if err != nil {
		return []byte{}, createCommunityRemoteBurnErrorResponse(err)
	}
	var tokenIds []*big.Int
	for _, tokenId := range params.CommunityParams.TokenIds {
		tokenIds = append(tokenIds, tokenId.ToInt())
	}
	return collectiblesABI.Pack("remoteBurn", tokenIds)
}

func (s *CommunityRemoteBurnProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createCommunityRemoteBurnErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunityRemoteBurnErrorResponse(err)
	}

	toAddress := params.CommunityParams.GetTokenContractAddress()
	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &toAddress,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createCommunityRemoteBurnErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunityRemoteBurnProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunityRemoteBurnProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunityRemoteBurnProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunityRemoteBurnProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *CommunityRemoteBurnProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.CommunityParams.GetAmount(), nil
}

func (s *CommunityRemoteBurnProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.CommunityParams.GetTokenContractAddress(), nil
}
