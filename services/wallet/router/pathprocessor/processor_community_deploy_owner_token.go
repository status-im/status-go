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

	communitytokens "github.com/status-im/status-go/internal/contracts/community-tokens"
	communitytokendeployer "github.com/status-im/status-go/internal/contracts/community-tokens/deployer"
	crypto2 "github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"

	communitytokendeployer2 "github.com/status-im/status-go/internal/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/internal/errors"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type CommunityDeployOwnerTokenProcessor struct {
	contractMaker   *communitytokens.CommunityTokensContractMaker
	ethClientGetter rpc.EthClientGetter
	transactor      transactions.TransactorIface
	deployerOverrides map[uint64]common.Address
}

func NewCommunityDeployOwnerTokenProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface, deployerOverrides map[uint64]common.Address) *CommunityDeployOwnerTokenProcessor {
	return &CommunityDeployOwnerTokenProcessor{
		contractMaker:   communitytokens.NewCommunityTokensContractMakerMaker(ethClientGetter),
		ethClientGetter: ethClientGetter,
		transactor:      transactor,
		deployerOverrides: deployerOverrides,
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
	if len(sig) != crypto2.SignatureLength {
		err = &errors.ErrorResponse{
			Code:    ErrIncorrectSignatureFormat.Code,
			Details: fmt.Sprintf(ErrIncorrectSignatureFormat.Details, len(sig), crypto2.SignatureLength),
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
	communityPubKey, err := crypto2.DecompressPubkey(decoded)
	if err != nil {
		return common.Address{}, err
	}
	return common.Address(crypto2.PubkeyToAddress(*communityPubKey)), nil
}

func prepareDeploymentSignatureStruct(signature string, communityID string, addressFrom common.Address) (communitytokendeployer2.CommunityTokenDeployerDeploymentSignature, error) {
	r, s, v, err := decodeSignature(common.FromHex(signature))
	if err != nil {
		return communitytokendeployer2.CommunityTokenDeployerDeploymentSignature{}, err
	}
	communityEthAddress, err := convert33BytesPubKeyToEthAddress(communityID)
	if err != nil {
		return communitytokendeployer2.CommunityTokenDeployerDeploymentSignature{}, err
	}
	communitySignature := communitytokendeployer2.CommunityTokenDeployerDeploymentSignature{
		V:        v,
		R:        r,
		S:        s,
		Deployer: addressFrom,
		Signer:   communityEthAddress,
	}
	return communitySignature, nil
}

func (s *CommunityDeployOwnerTokenProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	deployerABI, err := abi.JSON(strings.NewReader(communitytokendeployer2.CommunityTokenDeployerABI))
	if err != nil {
		return []byte{}, err
	}

	ownerTokenConfig := communitytokendeployer2.CommunityTokenDeployerTokenConfig{
		Name:    params.CommunityParams.OwnerTokenParameters.Name,
		Symbol:  params.CommunityParams.OwnerTokenParameters.Symbol,
		BaseURI: params.CommunityParams.OwnerTokenParameters.TokenURI,
	}

	masterTokenConfig := communitytokendeployer2.CommunityTokenDeployerTokenConfig{
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

func (s *CommunityDeployOwnerTokenProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	if params.TestsMode {
		return 0, ErrNoEstimationFound
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createCommunityDeployOwnerTokenErrorResponse(err)
	}

	ethClient, err := s.ethClientGetter.EthClient(params.FromChain.ChainID)
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
	if params.ToAddr != (common.Address{}) {
		return params.ToAddr, nil
	}

	return communitytokendeployer.ContractAddressWithOverrides(params.FromChain.ChainID, s.deployerOverrides)
}
