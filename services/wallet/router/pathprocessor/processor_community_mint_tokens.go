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
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type CommunityMintTokensProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunityMintTokensProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunityMintTokensProcessor {
	return &CommunityMintTokensProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunityMintTokensErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunityMintTokensName, err)
}

func (s *CommunityMintTokensProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunityMintTokensName
}

func (s *CommunityMintTokensProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunityMintTokensProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

// if we want to mint 2 tokens to addresses ["a", "b"] we need to mint
// twice to every address - we need to send to smart contract table ["a", "a", "b", "b"]
func multiplyWalletAddresses(amount *big.Int, walletAddresses []common.Address) (result []common.Address) {
	n := amount.Int64()
	for _, address := range walletAddresses {
		for i := int64(0); i < n; i++ {
			result = append(result, address)
		}
	}
	return
}

func prepareMintAssetsData(amount *big.Int, walletAddresses []common.Address) (result []*big.Int) {
	length := len(walletAddresses)
	for i := 0; i < length; i++ {
		result = append(result, new(big.Int).Set(amount))
	}
	return result
}

func (s *CommunityMintTokensProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	switch params.CommunityParams.GetTokenType() {
	case protobuf.CommunityTokenType_ERC721:
		destinationAddresses := multiplyWalletAddresses(params.CommunityParams.GetAmount(), params.CommunityParams.WalletAddresses)
		collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
		if err != nil {
			return []byte{}, err
		}
		return collectiblesABI.Pack("mintTo", destinationAddresses)
	case protobuf.CommunityTokenType_ERC20:
		destinationAmounts := prepareMintAssetsData(params.CommunityParams.GetAmount(), params.CommunityParams.WalletAddresses)
		assetsABI, err := abi.JSON(strings.NewReader(assets.AssetsABI))
		if err != nil {
			return []byte{}, err
		}
		return assetsABI.Pack("mintTo", params.CommunityParams.WalletAddresses, destinationAmounts)
	default:
		return nil, ErrCommunityTokenType
	}
}

func (s *CommunityMintTokensProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createENSReleaseErrorResponse(err)
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createCommunityMintTokensErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunityMintTokensErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createCommunityMintTokensErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunityMintTokensProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunityMintTokensProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunityMintTokensProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunityMintTokensProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *CommunityMintTokensProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.CommunityParams.GetAmount(), nil
}

func (s *CommunityMintTokensProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.CommunityParams.GetTokenContractAddress(), nil
}
