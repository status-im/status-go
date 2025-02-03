package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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

type CommunityBurnProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunityBurnProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunityBurnProcessor {
	return &CommunityBurnProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunityBurnErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunityBurnName, err)
}

func (s *CommunityBurnProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunityBurnName
}

func (s *CommunityBurnProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunityBurnProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *CommunityBurnProcessor) validateBurnAmount(ctx context.Context, params ProcessorInputParams) error {
	remainingSupply, err := s.remainingSupply(ctx, params)
	if err != nil {
		return err
	}
	if params.CommunityParams.GetAmount().Cmp(remainingSupply) > 1 {
		return ErrBurnAmountTooHigh
	}
	return nil
}

func (s *CommunityBurnProcessor) remainingSupply(ctx context.Context, params ProcessorInputParams) (*big.Int, error) {
	switch params.CommunityParams.GetTokenType() {
	case protobuf.CommunityTokenType_ERC721:
		return s.remainingCollectiblesSupply(ctx, params.FromChain.ChainID, params.CommunityParams.GetTokenContractAddress())
	case protobuf.CommunityTokenType_ERC20:
		return s.remainingAssetsSupply(ctx, params.FromChain.ChainID, params.CommunityParams.GetTokenContractAddress())
	default:
		return nil, ErrCommunityTokenType
	}
}

func (s *CommunityBurnProcessor) maxSupply(ctx context.Context, params ProcessorInputParams) (*big.Int, error) {
	switch params.CommunityParams.GetTokenType() {
	case protobuf.CommunityTokenType_ERC721:
		return s.maxSupplyCollectibles(ctx, params.FromChain.ChainID, params.CommunityParams.GetTokenContractAddress())
	case protobuf.CommunityTokenType_ERC20:
		return s.maxSupplyAssets(ctx, params.FromChain.ChainID, params.CommunityParams.GetTokenContractAddress())
	default:
		return nil, ErrCommunityTokenType
	}
}

// RemainingSupply = MaxSupply - MintedCount
func (s *CommunityBurnProcessor) remainingCollectiblesSupply(ctx context.Context, chainID uint64, contractAddress common.Address) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.contractMaker.NewCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	maxSupply, err := contractInst.MaxSupply(callOpts)
	if err != nil {
		return nil, err
	}
	mintedCount, err := contractInst.MintedCount(callOpts)
	if err != nil {
		return nil, err
	}
	res := big.NewInt(0)
	res.Sub(maxSupply, mintedCount)
	return res, nil
}

// RemainingSupply = MaxSupply - TotalSupply
func (s *CommunityBurnProcessor) remainingAssetsSupply(ctx context.Context, chainID uint64, contractAddress common.Address) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.contractMaker.NewAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	maxSupply, err := contractInst.MaxSupply(callOpts)
	if err != nil {
		return nil, err
	}
	totalSupply, err := contractInst.TotalSupply(callOpts)
	if err != nil {
		return nil, err
	}
	res := big.NewInt(0)
	res.Sub(maxSupply, totalSupply)
	return res, nil
}

func (s *CommunityBurnProcessor) maxSupplyCollectibles(ctx context.Context, chainID uint64, contractAddress common.Address) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.contractMaker.NewCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (s *CommunityBurnProcessor) maxSupplyAssets(ctx context.Context, chainID uint64, contractAddress common.Address) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.contractMaker.NewAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (s *CommunityBurnProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	ctx := context.Background()
	err := s.validateBurnAmount(ctx, params)
	if err != nil {
		return []byte{}, createCommunityBurnErrorResponse(err)
	}

	maxSupply, err := s.maxSupply(ctx, params)
	if err != nil {
		return []byte{}, createCommunityBurnErrorResponse(err)
	}
	newMaxSupply := big.NewInt(0)
	newMaxSupply.Sub(maxSupply, params.CommunityParams.GetAmount())

	switch params.CommunityParams.GetTokenType() {
	case protobuf.CommunityTokenType_ERC721:
		collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
		if err != nil {
			return []byte{}, createCommunityBurnErrorResponse(err)
		}
		return collectiblesABI.Pack("setMaxSupply", newMaxSupply)
	case protobuf.CommunityTokenType_ERC20:
		assetsABI, err := abi.JSON(strings.NewReader(assets.AssetsABI))
		if err != nil {
			return []byte{}, createCommunityBurnErrorResponse(err)
		}
		return assetsABI.Pack("setMaxSupply", newMaxSupply)
	default:
		return nil, ErrCommunityTokenType
	}
}

func (s *CommunityBurnProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunityBurnErrorResponse(err)
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
		return 0, createCommunityBurnErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunityBurnProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunityBurnProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunityBurnProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunityBurnProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *CommunityBurnProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.CommunityParams.GetAmount(), nil
}

func (s *CommunityBurnProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.CommunityParams.GetTokenContractAddress(), nil
}
