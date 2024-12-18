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
	"github.com/status-im/status-go/contracts/community-tokens/ownertoken"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type CommunitySetSignerPubKeyProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunitySetSignerPubKeyProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunitySetSignerPubKeyProcessor {
	return &CommunitySetSignerPubKeyProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunitySetSignerPubKeyErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName, err)
}

func (s *CommunitySetSignerPubKeyProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName
}

func (s *CommunitySetSignerPubKeyProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunitySetSignerPubKeyProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *CommunitySetSignerPubKeyProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	ownerTokenABI, err := abi.JSON(strings.NewReader(ownertoken.OwnerTokenABI))
	if err != nil {
		return []byte{}, err
	}

	return ownerTokenABI.Pack("setSignerPublicKey", common.FromHex(params.CommunityParams.SignerPubKey))
}

func (s *CommunitySetSignerPubKeyProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createCommunitySetSignerPubKeyErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunitySetSignerPubKeyErrorResponse(err)
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
		return 0, createCommunitySetSignerPubKeyErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunitySetSignerPubKeyProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunitySetSignerPubKeyProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunitySetSignerPubKeyProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunitySetSignerPubKeyProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *CommunitySetSignerPubKeyProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *CommunitySetSignerPubKeyProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.CommunityParams.GetTokenContractAddress(), nil
}
