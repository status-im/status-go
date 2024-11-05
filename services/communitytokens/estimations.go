package communitytokens

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type CommunityTokenFees struct {
	GasUnits      uint64                  `json:"gasUnits"`
	SuggestedFees *fees.SuggestedFeesGwei `json:"suggestedFees"`
}

func weiToGwei(val *big.Int) *big.Float {
	result := new(big.Float)
	result.SetInt(val)

	unit := new(big.Int)
	unit.SetInt64(params.GWei)

	return result.Quo(result, new(big.Float).SetInt(unit))
}

func gweiToWei(val *big.Float) *big.Int {
	res, _ := new(big.Float).Mul(val, big.NewFloat(1000000000)).Int(nil)
	return res
}

func (s *Service) deployOwnerTokenEstimate(ctx context.Context, chainID uint64, fromAddress string,
	ownerTokenParameters DeploymentParameters, masterTokenParameters DeploymentParameters,
	communityID string, signerPubKey string) (*CommunityTokenFees, error) {

	gasUnits, err := s.deployOwnerTokenGasUnits(ctx, chainID, fromAddress, ownerTokenParameters, masterTokenParameters,
		communityID, signerPubKey)
	if err != nil {
		return nil, err
	}

	deployerAddress, err := communitytokendeployer.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), &deployerAddress, gasUnits, chainID)
}

func (s *Service) deployCollectiblesEstimate(ctx context.Context, chainID uint64, fromAddress string) (*CommunityTokenFees, error) {
	gasUnits, err := s.deployCollectiblesGasUnits(ctx, chainID, fromAddress)
	if err != nil {
		return nil, err
	}
	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), nil, gasUnits, chainID)
}

func (s *Service) deployAssetsEstimate(ctx context.Context, chainID uint64, fromAddress string) (*CommunityTokenFees, error) {
	gasUnits, err := s.deployAssetsGasUnits(ctx, chainID, fromAddress)
	if err != nil {
		return nil, err
	}
	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), nil, gasUnits, chainID)
}

func (s *Service) mintTokensEstimate(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (*CommunityTokenFees, error) {
	gasUnits, err := s.mintTokensGasUnits(ctx, chainID, contractAddress, fromAddress, walletAddresses, amount)
	if err != nil {
		return nil, err
	}
	toAddress := common.HexToAddress(contractAddress)
	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), &toAddress, gasUnits, chainID)
}

func (s *Service) remoteBurnEstimate(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, tokenIds []*bigint.BigInt) (*CommunityTokenFees, error) {
	gasUnits, err := s.remoteBurnGasUnits(ctx, chainID, contractAddress, fromAddress, tokenIds)
	if err != nil {
		return nil, err
	}
	toAddress := common.HexToAddress(contractAddress)
	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), &toAddress, gasUnits, chainID)
}

func (s *Service) burnEstimate(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, burnAmount *bigint.BigInt) (*CommunityTokenFees, error) {
	gasUnits, err := s.burnGasUnits(ctx, chainID, contractAddress, fromAddress, burnAmount)
	if err != nil {
		return nil, err
	}
	toAddress := common.HexToAddress(contractAddress)
	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), &toAddress, gasUnits, chainID)
}

func (s *Service) setSignerPubKeyEstimate(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, newSignerPubKey string) (*CommunityTokenFees, error) {
	gasUnits, err := s.setSignerPubKeyGasUnits(ctx, chainID, contractAddress, fromAddress, newSignerPubKey)
	if err != nil {
		return nil, err
	}
	toAddress := common.HexToAddress(contractAddress)
	return s.prepareCommunityTokenFees(ctx, common.HexToAddress(fromAddress), &toAddress, gasUnits, chainID)
}

func (s *Service) setSignerPubKeyGasUnits(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, newSignerPubKey string) (uint64, error) {
	if len(newSignerPubKey) <= 0 {
		return 0, fmt.Errorf("signerPubKey is empty")
	}

	contractInst, err := s.NewOwnerTokenInstance(chainID, contractAddress)
	if err != nil {
		return 0, err
	}
	ownerTokenInstance := &OwnerTokenInstance{instance: contractInst}

	return s.estimateMethodForTokenInstance(ctx, ownerTokenInstance, chainID, contractAddress, fromAddress, "setSignerPublicKey", common.FromHex(newSignerPubKey))
}

func (s *Service) burnGasUnits(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, burnAmount *bigint.BigInt) (uint64, error) {
	err := s.validateBurnAmount(ctx, burnAmount, chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	newMaxSupply, err := s.prepareNewMaxSupply(ctx, chainID, contractAddress, burnAmount)
	if err != nil {
		return 0, err
	}

	return s.estimateMethod(ctx, chainID, contractAddress, fromAddress, "setMaxSupply", newMaxSupply)
}

func (s *Service) remoteBurnGasUnits(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, tokenIds []*bigint.BigInt) (uint64, error) {
	err := s.validateTokens(tokenIds)
	if err != nil {
		return 0, err
	}

	var tempTokenIds []*big.Int
	for _, v := range tokenIds {
		tempTokenIds = append(tempTokenIds, v.Int)
	}

	return s.estimateMethod(ctx, chainID, contractAddress, fromAddress, "remoteBurn", tempTokenIds)
}

func (s *Service) deployOwnerTokenGasUnits(ctx context.Context, chainID uint64, fromAddress string,
	ownerTokenParameters DeploymentParameters, masterTokenParameters DeploymentParameters,
	communityID string, signerPubKey string) (uint64, error) {
	ethClient, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return 0, err
	}

	deployerAddress, err := communitytokendeployer.ContractAddress(chainID)
	if err != nil {
		return 0, err
	}

	deployerABI, err := abi.JSON(strings.NewReader(communitytokendeployer.CommunityTokenDeployerABI))
	if err != nil {
		return 0, err
	}

	ownerTokenConfig := communitytokendeployer.CommunityTokenDeployerTokenConfig{
		Name:    ownerTokenParameters.Name,
		Symbol:  ownerTokenParameters.Symbol,
		BaseURI: ownerTokenParameters.TokenURI,
	}

	masterTokenConfig := communitytokendeployer.CommunityTokenDeployerTokenConfig{
		Name:    masterTokenParameters.Name,
		Symbol:  masterTokenParameters.Symbol,
		BaseURI: masterTokenParameters.TokenURI,
	}

	signature, err := s.Messenger.CreateCommunityTokenDeploymentSignature(ctx, chainID, fromAddress, communityID)
	if err != nil {
		return 0, err
	}

	communitySignature, err := prepareDeploymentSignatureStruct(types.HexBytes(signature).String(), communityID, common.HexToAddress(fromAddress))
	if err != nil {
		return 0, err
	}

	data, err := deployerABI.Pack("deploy", ownerTokenConfig, masterTokenConfig, communitySignature, common.FromHex(signerPubKey))

	if err != nil {
		return 0, err
	}

	toAddr := deployerAddress
	fromAddr := common.HexToAddress(fromAddress)

	callMsg := ethereum.CallMsg{
		From:  fromAddr,
		To:    &toAddr,
		Value: big.NewInt(0),
		Data:  data,
	}

	estimate, err := ethClient.EstimateGas(ctx, callMsg)
	if err != nil {
		return 0, err
	}

	finalEstimation := estimate + uint64(float32(estimate)*0.1)
	logutils.ZapLogger().Debug("Owner token deployment estimation", zap.Uint64("gas", finalEstimation))
	return finalEstimation, nil
}

func (s *Service) deployCollectiblesGasUnits(ctx context.Context, chainID uint64, fromAddress string) (uint64, error) {
	ethClient, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return 0, err
	}

	collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
	if err != nil {
		return 0, err
	}

	// use random parameters, they will not have impact on deployment results
	data, err := collectiblesABI.Pack("" /*constructor name is empty*/, "name", "SYMBOL", big.NewInt(20), true, false, "tokenUri",
		common.HexToAddress("0x77b48394c650520012795a1a25696d7eb542d110"), common.HexToAddress("0x77b48394c650520012795a1a25696d7eb542d110"))
	if err != nil {
		return 0, err
	}

	callMsg := ethereum.CallMsg{
		From:  common.HexToAddress(fromAddress),
		To:    nil,
		Value: big.NewInt(0),
		Data:  append(common.FromHex(collectibles.CollectiblesBin), data...),
	}
	estimate, err := ethClient.EstimateGas(ctx, callMsg)
	if err != nil {
		return 0, err
	}

	finalEstimation := estimate + uint64(float32(estimate)*0.1)
	logutils.ZapLogger().Debug("Collectibles deployment estimation", zap.Uint64("gas", finalEstimation))
	return finalEstimation, nil
}

func (s *Service) deployAssetsGasUnits(ctx context.Context, chainID uint64, fromAddress string) (uint64, error) {
	ethClient, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return 0, err
	}

	assetsABI, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return 0, err
	}

	// use random parameters, they will not have impact on deployment results
	data, err := assetsABI.Pack("" /*constructor name is empty*/, "name", "SYMBOL", uint8(18), big.NewInt(20), "tokenUri",
		common.HexToAddress("0x77b48394c650520012795a1a25696d7eb542d110"), common.HexToAddress("0x77b48394c650520012795a1a25696d7eb542d110"))
	if err != nil {
		return 0, err
	}

	callMsg := ethereum.CallMsg{
		From:  common.HexToAddress(fromAddress),
		To:    nil,
		Value: big.NewInt(0),
		Data:  append(common.FromHex(assets.AssetsBin), data...),
	}
	estimate, err := ethClient.EstimateGas(ctx, callMsg)
	if err != nil {
		return 0, err
	}

	finalEstimation := estimate + uint64(float32(estimate)*0.1)
	logutils.ZapLogger().Debug("Assets deployment estimation: ", zap.Uint64("gas", finalEstimation))
	return finalEstimation, nil
}

// if we want to mint 2 tokens to addresses ["a", "b"] we need to mint
// twice to every address - we need to send to smart contract table ["a", "a", "b", "b"]
func multiplyWalletAddresses(amount *bigint.BigInt, contractAddresses []string) []string {
	var totalAddresses []string
	for i := big.NewInt(1); i.Cmp(amount.Int) <= 0; {
		totalAddresses = append(totalAddresses, contractAddresses...)
		i.Add(i, big.NewInt(1))
	}
	return totalAddresses
}

func prepareMintCollectiblesData(walletAddresses []string, amount *bigint.BigInt) []common.Address {
	totalAddresses := multiplyWalletAddresses(amount, walletAddresses)
	var usersAddresses = []common.Address{}
	for _, k := range totalAddresses {
		usersAddresses = append(usersAddresses, common.HexToAddress(k))
	}
	return usersAddresses
}

func prepareMintAssetsData(walletAddresses []string, amount *bigint.BigInt) ([]common.Address, []*big.Int) {
	var usersAddresses = []common.Address{}
	var amountsList = []*big.Int{}
	for _, k := range walletAddresses {
		usersAddresses = append(usersAddresses, common.HexToAddress(k))
		amountsList = append(amountsList, amount.Int)
	}
	return usersAddresses, amountsList
}

func (s *Service) mintCollectiblesGasUnits(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	err := s.ValidateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return 0, err
	}
	usersAddresses := prepareMintCollectiblesData(walletAddresses, amount)
	return s.estimateMethod(ctx, chainID, contractAddress, fromAddress, "mintTo", usersAddresses)
}

func (s *Service) mintAssetsGasUnits(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	err := s.ValidateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return 0, err
	}
	usersAddresses, amountsList := prepareMintAssetsData(walletAddresses, amount)
	return s.estimateMethod(ctx, chainID, contractAddress, fromAddress, "mintTo", usersAddresses, amountsList)
}

func (s *Service) mintTokensGasUnits(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	tokenType, err := s.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return s.mintCollectiblesGasUnits(ctx, chainID, contractAddress, fromAddress, walletAddresses, amount)
	case protobuf.CommunityTokenType_ERC20:
		return s.mintAssetsGasUnits(ctx, chainID, contractAddress, fromAddress, walletAddresses, amount)
	default:
		return 0, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (s *Service) prepareCommunityTokenFees(ctx context.Context, from common.Address, to *common.Address, gasUnits uint64, chainID uint64) (*CommunityTokenFees, error) {
	suggestedFees, err := s.feeManager.SuggestedFeesGwei(ctx, chainID)
	if err != nil {
		return nil, err
	}

	txArgs := s.suggestedFeesToSendTxArgs(from, to, gasUnits, suggestedFees)

	l1Fee, err := s.estimateL1Fee(ctx, chainID, txArgs)
	if err == nil {
		suggestedFees.L1GasFee = weiToGwei(big.NewInt(int64(l1Fee)))
	}
	return &CommunityTokenFees{
		GasUnits:      gasUnits,
		SuggestedFees: suggestedFees,
	}, nil
}

func (s *Service) suggestedFeesToSendTxArgs(from common.Address, to *common.Address, gas uint64, suggestedFees *fees.SuggestedFeesGwei) wallettypes.SendTxArgs {
	sendArgs := wallettypes.SendTxArgs{}
	sendArgs.From = types.Address(from)
	sendArgs.To = (*types.Address)(to)
	sendArgs.Gas = (*hexutil.Uint64)(&gas)
	if suggestedFees.EIP1559Enabled {
		sendArgs.MaxPriorityFeePerGas = (*hexutil.Big)(gweiToWei(suggestedFees.MaxPriorityFeePerGas))
		sendArgs.MaxFeePerGas = (*hexutil.Big)(gweiToWei(suggestedFees.MaxFeePerGasMedium))
	} else {
		sendArgs.GasPrice = (*hexutil.Big)(gweiToWei(suggestedFees.GasPrice))
	}
	return sendArgs
}

func (s *Service) estimateL1Fee(ctx context.Context, chainID uint64, sendArgs wallettypes.SendTxArgs) (uint64, error) {
	transaction, _, err := s.transactor.ValidateAndBuildTransaction(chainID, sendArgs, -1)
	if err != nil {
		return 0, err
	}

	data, err := transaction.MarshalBinary()
	if err != nil {
		return 0, err
	}

	return s.feeManager.GetL1Fee(ctx, chainID, data)
}

func (s *Service) estimateMethodForTokenInstance(ctx context.Context, contractInstance TokenInstance, chainID uint64, contractAddress string, fromAddress string, methodName string, args ...interface{}) (uint64, error) {
	ethClient, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return 0, err
	}

	data, err := contractInstance.PackMethod(ctx, methodName, args...)

	if err != nil {
		return 0, err
	}

	toAddr := common.HexToAddress(contractAddress)
	fromAddr := common.HexToAddress(fromAddress)

	callMsg := ethereum.CallMsg{
		From:  fromAddr,
		To:    &toAddr,
		Value: big.NewInt(0),
		Data:  data,
	}
	estimate, err := ethClient.EstimateGas(ctx, callMsg)

	if err != nil {
		return 0, err
	}
	return estimate + uint64(float32(estimate)*0.1), nil
}

func (s *Service) estimateMethod(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, methodName string, args ...interface{}) (uint64, error) {
	contractInst, err := NewTokenInstance(s, chainID, contractAddress)
	if err != nil {
		return 0, err
	}
	return s.estimateMethodForTokenInstance(ctx, contractInst, chainID, contractAddress, fromAddress, methodName, args...)
}
