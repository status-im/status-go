package pathprocessor

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	communitytokens "github.com/status-im/status-go/contracts/community-tokens"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type CommunityDeployOwnerTokenProcessor struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
	transactor    transactions.TransactorIface
}

func NewCommunityDeployOwnerTokenProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface) *CommunityDeployOwnerTokenProcessor {
	return &CommunityDeployOwnerTokenProcessor{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
		transactor: transactor,
	}
}

func createCommunityDeployOwnerTokenErrorResponse(err error) error {
	return createErrorResponse(pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName, err)
}

func (s *CommunityDeployOwnerTokenProcessor) Name() string {
	return pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName
}

func (s *CommunityDeployOwnerTokenProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	return true, nil
}

func (s *CommunityDeployOwnerTokenProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func decodeSignature(sig []byte) (r [32]byte, s [32]byte, v uint8, err error) {
	if len(sig) != crypto.SignatureLength {
		err = &errors.ErrorResponse{
			Code:    ErrIncorrectSignatureFormat.Code,
			Details: fmt.Sprintf(ErrIncorrectSignatureFormat.Details, len(sig), crypto.SignatureLength),
		}
		return [32]byte{}, [32]byte{}, 0, err
	}
	copy(r[:], sig[:32])
	copy(s[:], sig[32:64])
	v = sig[64] + 27
	return r, s, v, nil
}

func convert33BytesPubKeyToEthAddress(pubKey string) (common.Address, error) {
	decoded, err := types.DecodeHex(pubKey)
	if err != nil {
		return common.Address{}, err
	}
	communityPubKey, err := crypto.DecompressPubkey(decoded)
	if err != nil {
		return common.Address{}, err
	}
	return common.Address(crypto.PubkeyToAddress(*communityPubKey)), nil
}

func prepareDeploymentSignatureStruct(signature string, communityID string, addressFrom common.Address) (communitytokendeployer.CommunityTokenDeployerDeploymentSignature, error) {
	r, s, v, err := decodeSignature(common.FromHex(signature))
	if err != nil {
		return communitytokendeployer.CommunityTokenDeployerDeploymentSignature{}, err
	}
	communityEthAddress, err := convert33BytesPubKeyToEthAddress(communityID)
	if err != nil {
		return communitytokendeployer.CommunityTokenDeployerDeploymentSignature{}, err
	}
	communitySignature := communitytokendeployer.CommunityTokenDeployerDeploymentSignature{
		V:        v,
		R:        r,
		S:        s,
		Deployer: addressFrom,
		Signer:   communityEthAddress,
	}
	return communitySignature, nil
}

func (s *CommunityDeployOwnerTokenProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	deployerABI, err := abi.JSON(strings.NewReader(communitytokendeployer.CommunityTokenDeployerABI))
	if err != nil {
		return []byte{}, err
	}

	ownerTokenConfig := communitytokendeployer.CommunityTokenDeployerTokenConfig{
		Name:    params.CommunityParams.OwnerTokenParameters.Name,
		Symbol:  params.CommunityParams.OwnerTokenParameters.Symbol,
		BaseURI: params.CommunityParams.OwnerTokenParameters.TokenURI,
	}

	masterTokenConfig := communitytokendeployer.CommunityTokenDeployerTokenConfig{
		Name:    params.CommunityParams.MasterTokenParameters.Name,
		Symbol:  params.CommunityParams.MasterTokenParameters.Symbol,
		BaseURI: params.CommunityParams.MasterTokenParameters.TokenURI,
	}

	communitySignature, err := prepareDeploymentSignatureStruct(params.CommunityParams.TokenDeploymentSignature,
		params.CommunityParams.CommunityID, params.FromAddr)
	if err != nil {
		return []byte{}, err
	}

	return deployerABI.Pack("deploy", ownerTokenConfig, masterTokenConfig, communitySignature, common.FromHex(params.CommunityParams.SignerPubKey))
}

func (s *CommunityDeployOwnerTokenProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createCommunityDeployOwnerTokenErrorResponse(err)
	}

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createCommunityDeployOwnerTokenErrorResponse(err)
	}

	ethClient, err := s.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createCommunityDeployOwnerTokenErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createCommunityDeployOwnerTokenErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor
	logutils.ZapLogger().Debug("CommunityDeployOwnerTokenProcessor estimation", zap.Uint64("gas", uint64(increasedEstimation)))

	return uint64(increasedEstimation), nil
}

func (s *CommunityDeployOwnerTokenProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, usedNonce uint64, err error) {
	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce, verifiedAccount)
}

func (s *CommunityDeployOwnerTokenProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, *sendArgs.TransferTx, lastUsedNonce)
}

func (s *CommunityDeployOwnerTokenProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	tx, n, e := s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
	if e != nil {
		return nil, 0, e
	}
	return tx, n, nil
}

func (s *CommunityDeployOwnerTokenProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	return params.AmountIn, nil
}

func (s *CommunityDeployOwnerTokenProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return communitytokendeployer.ContractAddress(params.FromChain.ChainID)
}
