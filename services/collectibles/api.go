package collectibles

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/contracts/community-tokens/mastertoken"
	"github.com/status-im/status-go/contracts/community-tokens/ownertoken"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/bigint"
	wcommon "github.com/status-im/status-go/services/wallet/common"
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
	ContractAddress string `json:"contractAddress"`
	TransactionHash string `json:"transactionHash"`
}

const maxSupply = 999999999

type DeploymentParameters struct {
	Name               string         `json:"name"`
	Symbol             string         `json:"symbol"`
	Supply             *bigint.BigInt `json:"supply"`
	InfiniteSupply     bool           `json:"infiniteSupply"`
	Transferable       bool           `json:"transferable"`
	RemoteSelfDestruct bool           `json:"remoteSelfDestruct"`
	TokenURI           string         `json:"tokenUri"`
	OwnerTokenAddress  string         `json:"ownerTokenAddress"`
	MasterTokenAddress string         `json:"masterTokenAddress"`
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

func (api *API) DeployCollectibles(ctx context.Context, chainID uint64, deploymentParameters DeploymentParameters, txArgs transactions.SendTxArgs, password string) (DeploymentDetails, error) {

	err := deploymentParameters.Validate(false)
	if err != nil {
		return DeploymentDetails{}, err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	address, tx, _, err := collectibles.DeployCollectibles(transactOpts, ethClient, deploymentParameters.Name,
		deploymentParameters.Symbol, deploymentParameters.GetSupply(),
		deploymentParameters.RemoteSelfDestruct, deploymentParameters.Transferable,
		deploymentParameters.TokenURI, common.HexToAddress(deploymentParameters.OwnerTokenAddress),
		common.HexToAddress(deploymentParameters.MasterTokenAddress))
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		transactions.DeployCommunityToken,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction error", "error", err)
		return DeploymentDetails{}, err
	}

	return DeploymentDetails{address.Hex(), tx.Hash().Hex()}, nil
}

func (api *API) DeployOwnerToken(ctx context.Context, chainID uint64, ownerTokenParameters DeploymentParameters, masterTokenParameters DeploymentParameters, txArgs transactions.SendTxArgs, password string) (DeploymentDetails, error) {
	err := ownerTokenParameters.Validate(false)
	if err != nil {
		return DeploymentDetails{}, err
	}

	err = masterTokenParameters.Validate(false)
	if err != nil {
		return DeploymentDetails{}, err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	signerPubKey := []byte{}

	address, tx, _, err := ownertoken.DeployOwnerToken(transactOpts, ethClient, ownerTokenParameters.Name,
		ownerTokenParameters.Symbol, ownerTokenParameters.TokenURI,
		masterTokenParameters.Name, masterTokenParameters.Symbol,
		masterTokenParameters.TokenURI, signerPubKey)
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		transactions.DeployOwnerToken,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction error", "error", err)
		return DeploymentDetails{}, err
	}

	return DeploymentDetails{address.Hex(), tx.Hash().Hex()}, nil
}

func (api *API) GetMasterTokenContractAddressFromHash(ctx context.Context, chainID uint64, txHash string) (string, error) {
	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return "", err
	}

	receipt, err := ethClient.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return "", err
	}

	logMasterTokenCreatedSig := []byte("MasterTokenCreated(address)")
	logMasterTokenCreatedSigHash := crypto.Keccak256Hash(logMasterTokenCreatedSig)

	for _, vLog := range receipt.Logs {
		if vLog.Topics[0].Hex() == logMasterTokenCreatedSigHash.Hex() {
			ownerTokenABI, err := abi.JSON(strings.NewReader(ownertoken.OwnerTokenABI))
			if err != nil {
				return "", err
			}
			event := new(ownertoken.OwnerTokenMasterTokenCreated)
			err = ownerTokenABI.UnpackIntoInterface(event, "MasterTokenCreated", vLog.Data)
			if err != nil {
				return "", err
			}
			return event.MasterToken.Hex(), nil
		}
	}
	return "", fmt.Errorf("can't find master token address in transaction: %v", txHash)
}

func (api *API) DeployAssets(ctx context.Context, chainID uint64, deploymentParameters DeploymentParameters, txArgs transactions.SendTxArgs, password string) (DeploymentDetails, error) {

	err := deploymentParameters.Validate(true)
	if err != nil {
		return DeploymentDetails{}, err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	address, tx, _, err := assets.DeployAssets(transactOpts, ethClient, deploymentParameters.Name,
		deploymentParameters.Symbol, deploymentParameters.GetSupply())
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	err = api.s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		transactions.DeployCommunityToken,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction error", "error", err)
		return DeploymentDetails{}, err
	}

	return DeploymentDetails{address.Hex(), tx.Hash().Hex()}, nil
}

// Returns gas units + 10%
func (api *API) DeployCollectiblesEstimate(ctx context.Context) (uint64, error) {
	gasAmount := uint64(2091605)
	return gasAmount + uint64(float32(gasAmount)*0.1), nil
}

// Returns gas units + 10%
func (api *API) DeployAssetsEstimate(ctx context.Context) (uint64, error) {
	gasAmount := uint64(957483)
	return gasAmount + uint64(float32(gasAmount)*0.1), nil
}

// Returns gas units + 10%
func (api *API) DeployOwnerTokenEstimate(ctx context.Context) (uint64, error) {
	ownerGasAmount := uint64(4389457)
	return ownerGasAmount + uint64(float32(ownerGasAmount)*0.1), nil
}

func (api *API) NewMasterTokenInstance(chainID uint64, contractAddress string) (*mastertoken.MasterToken, error) {
	backend, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return mastertoken.NewMasterToken(common.HexToAddress(contractAddress), backend)
}

func (api *API) NewCollectiblesInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	return api.s.manager.NewCollectiblesInstance(chainID, contractAddress)
}

func (api *API) NewAssetsInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	return api.s.manager.NewAssetsInstance(chainID, contractAddress)
}

// if we want to mint 2 tokens to addresses ["a", "b"] we need to mint
// twice to every address - we need to send to smart contract table ["a", "a", "b", "b"]
func (api *API) multiplyWalletAddresses(amount *bigint.BigInt, contractAddresses []string) []string {
	var totalAddresses []string
	for i := big.NewInt(1); i.Cmp(amount.Int) <= 0; {
		totalAddresses = append(totalAddresses, contractAddresses...)
		i.Add(i, big.NewInt(1))
	}
	return totalAddresses
}

func (api *API) PrepareMintCollectiblesData(walletAddresses []string, amount *bigint.BigInt) []common.Address {
	totalAddresses := api.multiplyWalletAddresses(amount, walletAddresses)
	var usersAddresses = []common.Address{}
	for _, k := range totalAddresses {
		usersAddresses = append(usersAddresses, common.HexToAddress(k))
	}
	return usersAddresses
}

// Universal minting function for every type of token.
func (api *API) MintTokens(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, walletAddresses []string, amount *bigint.BigInt) (string, error) {

	err := api.ValidateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	contractInst, err := NewTokenInstance(api, chainID, contractAddress)
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
		transactions.AirdropCommunityToken,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction error", "error", err)
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func (api *API) EstimateMintTokens(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	tokenType, err := api.s.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return api.EstimateMintCollectibles(ctx, chainID, contractAddress, fromAddress, walletAddresses, amount)
	case protobuf.CommunityTokenType_ERC20:
		return api.EstimateMintAssets(ctx, chainID, contractAddress, fromAddress, walletAddresses, amount)
	default:
		return 0, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (api *API) EstimateMintCollectibles(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	err := api.ValidateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return 0, err
	}
	usersAddresses := api.PrepareMintCollectiblesData(walletAddresses, amount)
	return api.estimateMethod(ctx, chainID, contractAddress, fromAddress, "mintTo", usersAddresses)
}

func (api *API) PrepareMintAssetsData(walletAddresses []string, amount *bigint.BigInt) ([]common.Address, []*big.Int) {
	var usersAddresses = []common.Address{}
	var amountsList = []*big.Int{}
	for _, k := range walletAddresses {
		usersAddresses = append(usersAddresses, common.HexToAddress(k))
		amountsList = append(amountsList, amount.Int)
	}
	return usersAddresses, amountsList
}

// Estimate MintAssets cost.
func (api *API) EstimateMintAssets(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	err := api.ValidateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return 0, err
	}
	usersAddresses, amountsList := api.PrepareMintAssetsData(walletAddresses, amount)
	return api.estimateMethod(ctx, chainID, contractAddress, fromAddress, "mintTo", usersAddresses, amountsList)
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
func (api *API) RemoteBurn(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, tokenIds []*bigint.BigInt) (string, error) {
	err := api.validateTokens(tokenIds)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	var tempTokenIds []*big.Int
	for _, v := range tokenIds {
		tempTokenIds = append(tempTokenIds, v.Int)
	}

	contractInst, err := NewTokenInstance(api, chainID, contractAddress)
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
		transactions.RemoteDestructCollectible,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction error", "error", err)
		return "", err
	}

	return tx.Hash().Hex(), nil
}

// This is only ERC721 function
func (api *API) EstimateRemoteBurn(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, tokenIds []*bigint.BigInt) (uint64, error) {
	err := api.validateTokens(tokenIds)
	if err != nil {
		return 0, err
	}

	var tempTokenIds []*big.Int
	for _, v := range tokenIds {
		tempTokenIds = append(tempTokenIds, v.Int)
	}

	return api.estimateMethod(ctx, chainID, contractAddress, fromAddress, "remoteBurn", tempTokenIds)
}

func (api *API) GetCollectiblesContractInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	return api.s.manager.GetCollectiblesContractInstance(chainID, contractAddress)
}

func (api *API) GetAssetContractInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	return api.s.manager.GetAssetContractInstance(chainID, contractAddress)
}

func (api *API) RemainingSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	tokenType, err := api.s.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return api.remainingCollectiblesSupply(ctx, chainID, contractAddress)
	case protobuf.CommunityTokenType_ERC20:
		return api.remainingAssetsSupply(ctx, chainID, contractAddress)
	default:
		return nil, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

// RemainingSupply = MaxSupply - MintedCount
func (api *API) remainingCollectiblesSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.NewCollectiblesInstance(chainID, contractAddress)
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
	var res = new(big.Int)
	res.Sub(maxSupply, mintedCount)
	return &bigint.BigInt{Int: res}, nil
}

// RemainingSupply = MaxSupply - TotalSupply
func (api *API) remainingAssetsSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.NewAssetsInstance(chainID, contractAddress)
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
	var res = new(big.Int)
	res.Sub(maxSupply, totalSupply)
	return &bigint.BigInt{Int: res}, nil
}

func (api *API) maxSupplyCollectibles(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.NewCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (api *API) maxSupplyAssets(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.NewAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (api *API) maxSupply(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	tokenType, err := api.s.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return nil, err
	}

	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return api.maxSupplyCollectibles(ctx, chainID, contractAddress)
	case protobuf.CommunityTokenType_ERC20:
		return api.maxSupplyAssets(ctx, chainID, contractAddress)
	default:
		return nil, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (api *API) prepareNewMaxSupply(ctx context.Context, chainID uint64, contractAddress string, burnAmount *bigint.BigInt) (*big.Int, error) {
	maxSupply, err := api.maxSupply(ctx, chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	var newMaxSupply = new(big.Int)
	newMaxSupply.Sub(maxSupply, burnAmount.Int)
	return newMaxSupply, nil
}

func (api *API) Burn(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, burnAmount *bigint.BigInt) (string, error) {
	err := api.validateBurnAmount(ctx, burnAmount, chainID, contractAddress)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.s.accountsManager, api.s.config.KeyStoreDir, txArgs.From, password))

	newMaxSupply, err := api.prepareNewMaxSupply(ctx, chainID, contractAddress, burnAmount)
	if err != nil {
		return "", err
	}

	contractInst, err := NewTokenInstance(api, chainID, contractAddress)
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
		transactions.BurnCommunityToken,
		transactions.AutoDelete,
	)
	if err != nil {
		log.Error("TrackPendingTransaction error", "error", err)
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func (api *API) EstimateBurn(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, burnAmount *bigint.BigInt) (uint64, error) {
	err := api.validateBurnAmount(ctx, burnAmount, chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	newMaxSupply, err := api.prepareNewMaxSupply(ctx, chainID, contractAddress, burnAmount)
	if err != nil {
		return 0, err
	}

	return api.estimateMethod(ctx, chainID, contractAddress, fromAddress, "setMaxSupply", newMaxSupply)
}

func (api *API) ValidateWalletsAndAmounts(walletAddresses []string, amount *bigint.BigInt) error {
	if len(walletAddresses) == 0 {
		return errors.New("wallet addresses list is empty")
	}
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return errors.New("amount is <= 0")
	}
	return nil
}

func (api *API) validateTokens(tokenIds []*bigint.BigInt) error {
	if len(tokenIds) == 0 {
		return errors.New("token list is empty")
	}
	return nil
}

func (api *API) validateBurnAmount(ctx context.Context, burnAmount *bigint.BigInt, chainID uint64, contractAddress string) error {
	if burnAmount.Cmp(big.NewInt(0)) <= 0 {
		return errors.New("burnAmount is less than 0")
	}
	remainingSupply, err := api.RemainingSupply(ctx, chainID, contractAddress)
	if err != nil {
		return err
	}
	if burnAmount.Cmp(remainingSupply.Int) > 1 {
		return errors.New("burnAmount is bigger than remaining amount")
	}
	return nil
}

func (api *API) estimateMethod(ctx context.Context, chainID uint64, contractAddress string, fromAddress string, methodName string, args ...interface{}) (uint64, error) {
	ethClient, err := api.s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return 0, err
	}

	contractInst, err := NewTokenInstance(api, chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	data, err := contractInst.PackMethod(ctx, methodName, args...)

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
