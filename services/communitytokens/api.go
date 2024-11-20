package communitytokens

import (
	"context"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/contracts/community-tokens/ownertoken"
	communityownertokenregistry "github.com/status-im/status-go/contracts/community-tokens/registry"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/bigint"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(s *Service) *API {
	return &API{
		s: s,
	}
}

type API struct {
	s *Service
}

type DeploymentDetails struct {
	ContractAddress string                `json:"contractAddress"`
	TransactionHash string                `json:"transactionHash"`
	CommunityToken  *token.CommunityToken `json:"communityToken"`
	OwnerToken      *token.CommunityToken `json:"ownerToken"`
	MasterToken     *token.CommunityToken `json:"masterToken"`
}

const maxSupply = 999999999

type DeploymentParameters struct {
	Name               string               `json:"name"`
	Symbol             string               `json:"symbol"`
	Supply             *bigint.BigInt       `json:"supply"`
	InfiniteSupply     bool                 `json:"infiniteSupply"`
	Transferable       bool                 `json:"transferable"`
	RemoteSelfDestruct bool                 `json:"remoteSelfDestruct"`
	TokenURI           string               `json:"tokenUri"`
	OwnerTokenAddress  string               `json:"ownerTokenAddress"`
	MasterTokenAddress string               `json:"masterTokenAddress"`
	CommunityID        string               `json:"communityId"`
	Description        string               `json:"description"`
	CroppedImage       *images.CroppedImage `json:"croppedImage,omitempty"` // for community tokens
	Base64Image        string               `json:"base64image"`            // for owner & master tokens
	Decimals           int                  `json:"decimals"`
}

func (d *DeploymentParameters) GetSupply() *big.Int {
	if d.InfiniteSupply {
		return d.GetInfiniteSupply()
	}
	return d.Supply.Int
}

// infinite supply for ERC721 is 2^256-1
func (d *DeploymentParameters) GetInfiniteSupply() *big.Int {
	return GetInfiniteSupply()
}

func GetInfiniteSupply() *big.Int {
	max := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	max.Sub(max, big.NewInt(1))
	return max
}

func (d *DeploymentParameters) Validate(isAsset bool) error {
	if len(d.Name) <= 0 {
		return errors.New("empty collectible name")
	}
	if len(d.Symbol) <= 0 {
		return errors.New("empty collectible symbol")
	}
	var maxForType = big.NewInt(maxSupply)
	if isAsset {
		assetMultiplier, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		maxForType = maxForType.Mul(maxForType, assetMultiplier)
	}
	if !d.InfiniteSupply && (d.Supply.Cmp(big.NewInt(0)) < 0 || d.Supply.Cmp(maxForType) > 0) {
		return fmt.Errorf("wrong supply value: %v", d.Supply)
	}
	return nil
}

func (api *API) DeployCollectibles(ctx context.Context, chainID uint64, deploymentParameters DeploymentParameters, txArgs wallettypes.SendTxArgs, password string) (DeploymentDetails, error) {
	err := deploymentParameters.Validate(false)
	if err != nil {
		return DeploymentDetails{}, err
	}
	transactOpts := txArgs.ToTransactOpts(utils.VerifyPasswordAndGetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return DeploymentDetails{}, err
	}
	address, tx, _, err := collectibles.DeployCollectibles(transactOpts, ethClient, deploymentParameters.Name,
		deploymentParameters.Symbol, deploymentParameters.GetSupply(),
		deploymentParameters.RemoteSelfDestruct, deploymentParameters.Transferable,
		deploymentParameters.TokenURI, common.HexToAddress(deploymentParameters.OwnerTokenAddress),
		common.HexToAddress(deploymentParameters.MasterTokenAddress))
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return DeploymentDetails{}, err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		address,
		transactions.DeployCommunityToken,
		transactions.Keep,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return DeploymentDetails{}, err
	}

	savedCommunityToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), deploymentParameters, txArgs.From.Hex(), address.Hex(),
		protobuf.CommunityTokenType_ERC721, token.CommunityLevel, tx.Hash().Hex())
	if err != nil {
		return DeploymentDetails{}, err
	}

	return DeploymentDetails{
		ContractAddress: address.Hex(),
		TransactionHash: tx.Hash().Hex(),
		CommunityToken:  savedCommunityToken}, nil
}

func decodeSignature(sig []byte) (r [32]byte, s [32]byte, v uint8, err error) {
	if len(sig) != crypto.SignatureLength {
		return [32]byte{}, [32]byte{}, 0, fmt.Errorf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength)
	}
	copy(r[:], sig[:32])
	copy(s[:], sig[32:64])
	v = sig[64] + 27
	return r, s, v, nil
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

func (api *API) DeployOwnerToken(ctx context.Context, chainID uint64,
	ownerTokenParameters DeploymentParameters, masterTokenParameters DeploymentParameters,
	signerPubKey string, txArgs wallettypes.SendTxArgs, password string) (DeploymentDetails, error) {
	err := ownerTokenParameters.Validate(false)
	if err != nil {
		return DeploymentDetails{}, err
	}

	if len(signerPubKey) <= 0 {
		return DeploymentDetails{}, fmt.Errorf("signerPubKey is empty")
	}

	err = masterTokenParameters.Validate(false)
	if err != nil {
		return DeploymentDetails{}, err
	}

	transactOpts := txArgs.ToTransactOpts(utils.VerifyPasswordAndGetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	deployerContractInst, err := api.NewCommunityTokenDeployerInstance(chainID)
	if err != nil {
		return DeploymentDetails{}, err
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

	signature, err := api.s.Messenger.CreateCommunityTokenDeploymentSignature(context.Background(), chainID, txArgs.From.Hex(), ownerTokenParameters.CommunityID)
	if err != nil {
		return DeploymentDetails{}, err
	}

	communitySignature, err := prepareDeploymentSignatureStruct(types.HexBytes(signature).String(), ownerTokenParameters.CommunityID, common.Address(txArgs.From))
	if err != nil {
		return DeploymentDetails{}, err
	}

	logutils.ZapLogger().Debug("Prepare deployment", zap.Any("signature", communitySignature))

	tx, err := deployerContractInst.Deploy(transactOpts, ownerTokenConfig, masterTokenConfig, communitySignature, common.FromHex(signerPubKey))

	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return DeploymentDetails{}, err
	}

	logutils.ZapLogger().Debug("Contract deployed", zap.Stringer("hash", tx.Hash()))

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		common.Address{},
		transactions.DeployOwnerToken,
		transactions.Keep,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return DeploymentDetails{}, err
	}

	savedOwnerToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), ownerTokenParameters, txArgs.From.Hex(),
		api.s.TemporaryOwnerContractAddress(tx.Hash().Hex()), protobuf.CommunityTokenType_ERC721, token.OwnerLevel, tx.Hash().Hex())
	if err != nil {
		return DeploymentDetails{}, err
	}
	savedMasterToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), masterTokenParameters, txArgs.From.Hex(),
		api.s.TemporaryMasterContractAddress(tx.Hash().Hex()), protobuf.CommunityTokenType_ERC721, token.MasterLevel, tx.Hash().Hex())
	if err != nil {
		return DeploymentDetails{}, err
	}

	return DeploymentDetails{
		ContractAddress: "",
		TransactionHash: tx.Hash().Hex(),
		OwnerToken:      savedOwnerToken,
		MasterToken:     savedMasterToken}, nil
}

// recovery function which starts transaction tracking again
func (api *API) ReTrackOwnerTokenDeploymentTransaction(ctx context.Context, chainID uint64, contractAddress string) error {
	return api.s.ReTrackOwnerTokenDeploymentTransaction(ctx, chainID, contractAddress)
}

func (api *API) DeployAssets(ctx context.Context, chainID uint64, deploymentParameters DeploymentParameters, txArgs wallettypes.SendTxArgs, password string) (DeploymentDetails, error) {

	err := deploymentParameters.Validate(true)
	if err != nil {
		return DeploymentDetails{}, err
	}

	transactOpts := txArgs.ToTransactOpts(utils.VerifyPasswordAndGetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return DeploymentDetails{}, err
	}

	const decimals = 18
	address, tx, _, err := assets.DeployAssets(transactOpts, ethClient, deploymentParameters.Name,
		deploymentParameters.Symbol, decimals, deploymentParameters.GetSupply(),
		deploymentParameters.TokenURI,
		common.HexToAddress(deploymentParameters.OwnerTokenAddress),
		common.HexToAddress(deploymentParameters.MasterTokenAddress))
	if err != nil {
		logutils.ZapLogger().Error(err.Error())
		return DeploymentDetails{}, err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		address,
		transactions.DeployCommunityToken,
		transactions.Keep,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return DeploymentDetails{}, err
	}

	savedCommunityToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), deploymentParameters, txArgs.From.Hex(), address.Hex(),
		protobuf.CommunityTokenType_ERC20, token.CommunityLevel, tx.Hash().Hex())
	if err != nil {
		return DeploymentDetails{}, err
	}

	return DeploymentDetails{
		ContractAddress: address.Hex(),
		TransactionHash: tx.Hash().Hex(),
		CommunityToken:  savedCommunityToken}, nil
}

func (api *API) DeployCollectiblesEstimate(ctx context.Context, chainID uint64, fromAddress string) (*CommunityTokenFees, error) {
	return api.s.deployCollectiblesEstimate(ctx, chainID, fromAddress)
}

func (api *API) DeployAssetsEstimate(ctx context.Context, chainID uint64, fromAddress string) (*CommunityTokenFees, error) {
	return api.s.deployAssetsEstimate(ctx, chainID, fromAddress)
}

func (api *API) DeployOwnerTokenEstimate(ctx context.Context, chainID uint64, fromAddress string,
	ownerTokenParameters DeploymentParameters, masterTokenParameters DeploymentParameters,
	communityID string, signerPubKey string) (*CommunityTokenFees, error) {
	return api.s.deployOwnerTokenEstimate(ctx, chainID, fromAddress, ownerTokenParameters, masterTokenParameters, communityID, signerPubKey)
}

func (api *API) EstimateMintTokens(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (*CommunityTokenFees, error) {
	return api.s.mintTokensEstimate(ctx, chainID, contractAddress, fromAddress, walletAddresses, amount)
}

// This is only ERC721 function
func (api *API) EstimateRemoteBurn(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, tokenIds []*bigint.BigInt) (*CommunityTokenFees, error) {
	return api.s.remoteBurnEstimate(ctx, chainID, contractAddress, fromAddress, tokenIds)
}

func (api *API) EstimateBurn(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, burnAmount *bigint.BigInt) (*CommunityTokenFees, error) {
	return api.s.burnEstimate(ctx, chainID, contractAddress, fromAddress, burnAmount)
}

func (api *API) EstimateSetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, newSignerPubKey string) (*CommunityTokenFees, error) {
	return api.s.setSignerPubKeyEstimate(ctx, chainID, contractAddress, fromAddress, newSignerPubKey)
}

func (api *API) NewOwnerTokenInstance(chainID uint64, contractAddress string) (*ownertoken.OwnerToken, error) {
	return api.s.NewOwnerTokenInstance(chainID, contractAddress)
}

func (api *API) NewCommunityTokenDeployerInstance(chainID uint64) (*communitytokendeployer.CommunityTokenDeployer, error) {
	return api.s.manager.NewCommunityTokenDeployerInstance(chainID)
}

func (api *API) NewCommunityOwnerTokenRegistryInstance(chainID uint64, contractAddress string) (*communityownertokenregistry.CommunityOwnerTokenRegistry, error) {
	return api.s.NewCommunityOwnerTokenRegistryInstance(chainID, contractAddress)
}

func (api *API) NewCollectiblesInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	return api.s.manager.NewCollectiblesInstance(chainID, contractAddress)
}

func (api *API) NewAssetsInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	return api.s.manager.NewAssetsInstance(chainID, contractAddress)
}

// Universal minting function for every type of token.
func (api *API) MintTokens(ctx context.Context, chainID uint64, contractAddress string, txArgs wallettypes.SendTxArgs, password string, walletAddresses []string, amount *bigint.BigInt) (string, error) {

	err := api.s.ValidateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.VerifyPasswordAndGetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	contractInst, err := NewTokenInstance(api.s, chainID, contractAddress)
	if err != nil {
		return "", err
	}

	tx, err := contractInst.Mint(transactOpts, walletAddresses, amount)
	if err != nil {
		return "", err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		common.HexToAddress(contractAddress),
		transactions.AirdropCommunityToken,
		transactions.Keep,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	return tx.Hash().Hex(), nil
}

// This is only ERC721 function
func (api *API) RemoteDestructedAmount(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.NewCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}

	// total supply = airdropped only (w/o burnt)
	totalSupply, err := contractInst.TotalSupply(callOpts)
	if err != nil {
		return nil, err
	}

	// minted = all created tokens (airdropped and remotely destructed)
	mintedCount, err := contractInst.MintedCount(callOpts)
	if err != nil {
		return nil, err
	}

	var res = new(big.Int)
	res.Sub(mintedCount, totalSupply)

	return &bigint.BigInt{Int: res}, nil
}

// This is only ERC721 function
func (api *API) RemoteBurn(ctx context.Context, chainID uint64, contractAddress string, txArgs wallettypes.SendTxArgs, password string, tokenIds []*bigint.BigInt, additionalData string) (string, error) {
	err := api.s.validateTokens(tokenIds)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.VerifyPasswordAndGetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	var tempTokenIds []*big.Int
	for _, v := range tokenIds {
		tempTokenIds = append(tempTokenIds, v.Int)
	}

	contractInst, err := NewTokenInstance(api.s, chainID, contractAddress)
	if err != nil {
		return "", err
	}

	tx, err := contractInst.RemoteBurn(transactOpts, tempTokenIds)
	if err != nil {
		return "", err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		common.HexToAddress(contractAddress),
		transactions.RemoteDestructCollectible,
		transactions.Keep,
		additionalData,
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func (api *API) GetCollectiblesContractInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	return api.s.manager.GetCollectiblesContractInstance(chainID, contractAddress)
}

func (api *API) GetAssetContractInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	return api.s.manager.GetAssetContractInstance(chainID, contractAddress)
}

func (api *API) RemainingSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	return api.s.remainingSupply(ctx, chainID, contractAddress)
}

func (api *API) Burn(ctx context.Context, chainID uint64, contractAddress string, txArgs wallettypes.SendTxArgs, password string, burnAmount *bigint.BigInt) (string, error) {
	err := api.s.validateBurnAmount(ctx, burnAmount, chainID, contractAddress)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.VerifyPasswordAndGetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	newMaxSupply, err := api.s.prepareNewMaxSupply(ctx, chainID, contractAddress, burnAmount)
	if err != nil {
		return "", err
	}

	contractInst, err := NewTokenInstance(api.s, chainID, contractAddress)
	if err != nil {
		return "", err
	}

	tx, err := contractInst.SetMaxSupply(transactOpts, newMaxSupply)
	if err != nil {
		return "", err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		common.HexToAddress(contractAddress),
		transactions.BurnCommunityToken,
		transactions.Keep,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	return tx.Hash().Hex(), nil
}

// Gets signer public key from smart contract with a given chainId and address
func (api *API) GetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string) (string, error) {
	return api.s.GetSignerPubKey(ctx, chainID, contractAddress)
}

// Gets signer public key directly from deployer contract
func (api *API) SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error) {
	return api.s.SafeGetSignerPubKey(ctx, chainID, communityID)
}

// Gets owner token contract address from deployer contract
func (api *API) SafeGetOwnerTokenAddress(ctx context.Context, chainID uint64, communityID string) (string, error) {
	return api.s.SafeGetOwnerTokenAddress(ctx, chainID, communityID)
}

func (api *API) SetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string, txArgs wallettypes.SendTxArgs, password string, newSignerPubKey string) (string, error) {
	return api.s.SetSignerPubKey(ctx, chainID, contractAddress, txArgs, password, newSignerPubKey)
}

func (api *API) OwnerTokenOwnerAddress(ctx context.Context, chainID uint64, contractAddress string) (string, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.NewOwnerTokenInstance(chainID, contractAddress)
	if err != nil {
		return "", err
	}
	ownerAddress, err := contractInst.OwnerOf(callOpts, big.NewInt(0))
	if err != nil {
		return "", err
	}
	return ownerAddress.Hex(), nil
}
