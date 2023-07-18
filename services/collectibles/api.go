package collectibles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/assets"
	"github.com/status-im/status-go/contracts/collectibles"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(rpcClient *rpc.Client, accountsManager *account.GethManager, rpcFiltersSrvc *rpcfilters.Service, config *params.NodeConfig, appDb *sql.DB) *API {
	return &API{
		RPCClient:       rpcClient,
		accountsManager: accountsManager,
		rpcFiltersSrvc:  rpcFiltersSrvc,
		config:          config,
		db:              NewCommunityTokensDatabase(appDb),
	}
}

type API struct {
	RPCClient       *rpc.Client
	accountsManager *account.GethManager
	rpcFiltersSrvc  *rpcfilters.Service
	config          *params.NodeConfig
	db              *Database
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
}

func (d *DeploymentParameters) GetSupply() *big.Int {
	if d.InfiniteSupply {
		return d.GetInfiniteSupply()
	}
	return d.Supply.Int
}

// infinite supply for ERC721 is 2^256-1
func (d *DeploymentParameters) GetInfiniteSupply() *big.Int {
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

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.RPCClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	address, tx, _, err := collectibles.DeployCollectibles(transactOpts, ethClient, deploymentParameters.Name,
		deploymentParameters.Symbol, deploymentParameters.GetSupply(),
		deploymentParameters.RemoteSelfDestruct, deploymentParameters.Transferable,
		deploymentParameters.TokenURI)
	if err != nil {
		log.Error(err.Error())
		return DeploymentDetails{}, err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(&rpcfilters.PendingTxInfo{
		Hash:    tx.Hash(),
		Type:    string(transactions.DeployCommunityToken),
		From:    common.Address(txArgs.From),
		ChainID: chainID,
	})

	return DeploymentDetails{address.Hex(), tx.Hash().Hex()}, nil
}

func (api *API) DeployAssets(ctx context.Context, chainID uint64, deploymentParameters DeploymentParameters, txArgs transactions.SendTxArgs, password string) (DeploymentDetails, error) {

	err := deploymentParameters.Validate(true)
	if err != nil {
		return DeploymentDetails{}, err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password))

	ethClient, err := api.RPCClient.EthClient(chainID)
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

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(&rpcfilters.PendingTxInfo{
		Hash:    tx.Hash(),
		Type:    string(transactions.DeployCommunityToken),
		From:    common.Address(txArgs.From),
		ChainID: chainID,
	})

	return DeploymentDetails{address.Hex(), tx.Hash().Hex()}, nil
}

// Returns gas units + 10%
func (api *API) DeployCollectiblesEstimate(ctx context.Context) (uint64, error) {
	gasAmount := uint64(1960645)
	return gasAmount + uint64(float32(gasAmount)*0.1), nil
}

// Returns gas units + 10%
func (api *API) DeployAssetsEstimate(ctx context.Context) (uint64, error) {
	gasAmount := uint64(957483)
	return gasAmount + uint64(float32(gasAmount)*0.1), nil
}

func (api *API) newCollectiblesInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	backend, err := api.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return collectibles.NewCollectibles(common.HexToAddress(contractAddress), backend)
}

func (api *API) newAssetsInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	backend, err := api.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return assets.NewAssets(common.HexToAddress(contractAddress), backend)
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

func (api *API) prepareMintCollectiblesData(walletAddresses []string, amount *bigint.BigInt) []common.Address {
	totalAddresses := api.multiplyWalletAddresses(amount, walletAddresses)
	var usersAddresses = []common.Address{}
	for _, k := range totalAddresses {
		usersAddresses = append(usersAddresses, common.HexToAddress(k))
	}
	return usersAddresses
}

// Universal minting function for both assets and collectibles.
// Checks contract type and runs MintCollectibles or MintAssets function.
func (api *API) MintTokens(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, walletAddresses []string, amount *bigint.BigInt) (string, error) {
	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return "", err
	}
	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return api.MintCollectibles(ctx, chainID, contractAddress, txArgs, password, walletAddresses, amount)
	case protobuf.CommunityTokenType_ERC20:
		return api.MintAssets(ctx, chainID, contractAddress, txArgs, password, walletAddresses, amount)
	default:
		return "", fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (api *API) EstimateMintTokens(ctx context.Context, chainID uint64, contractAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return 0, err
	}
	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return api.EstimateMintCollectibles(ctx, chainID, contractAddress, walletAddresses, amount)
	case protobuf.CommunityTokenType_ERC20:
		return api.EstimateMintAssets(ctx, chainID, contractAddress, walletAddresses, amount)
	default:
		return 0, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

// Create the amounty of collectible tokens and distribute them to all walletAddresses.
func (api *API) MintCollectibles(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, walletAddresses []string, amount *bigint.BigInt) (string, error) {
	err := api.validateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return "", err
	}

	contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return "", err
	}

	usersAddresses := api.prepareMintCollectiblesData(walletAddresses, amount)

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password))

	tx, err := contractInst.MintTo(transactOpts, usersAddresses)
	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(&rpcfilters.PendingTxInfo{
		Hash:    tx.Hash(),
		Type:    string(transactions.AirdropCommunityToken),
		From:    common.Address(txArgs.From),
		ChainID: chainID,
	})

	return tx.Hash().Hex(), nil
}

func (api *API) EstimateMintCollectibles(ctx context.Context, chainID uint64, contractAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	err := api.validateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return 0, err
	}
	usersAddresses := api.prepareMintCollectiblesData(walletAddresses, amount)
	return api.estimateMethod(ctx, chainID, contractAddress, "mintTo", usersAddresses)
}

func (api *API) prepareMintAssetsData(walletAddresses []string, amount *bigint.BigInt) ([]common.Address, []*big.Int) {
	var usersAddresses = []common.Address{}
	var amountsList = []*big.Int{}
	for _, k := range walletAddresses {
		usersAddresses = append(usersAddresses, common.HexToAddress(k))
		amountsList = append(amountsList, amount.Int)
	}
	return usersAddresses, amountsList
}

// Create the amount of assets tokens and distribute them to all walletAddresses.
// The amount should be in smallest denomination of the asset (like wei) with decimal = 18, eg.
// if we want to mint 2.34 of the token, then amount should be 234{16 zeros}.
func (api *API) MintAssets(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, walletAddresses []string, amount *bigint.BigInt) (string, error) {
	err := api.validateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return "", err
	}

	contractInst, err := api.newAssetsInstance(chainID, contractAddress)
	if err != nil {
		return "", err
	}

	usersAddresses, amountsList := api.prepareMintAssetsData(walletAddresses, amount)

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password))

	tx, err := contractInst.MintTo(transactOpts, usersAddresses, amountsList)
	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(&rpcfilters.PendingTxInfo{
		Hash:    tx.Hash(),
		Type:    string(transactions.AirdropCommunityToken),
		From:    common.Address(txArgs.From),
		ChainID: chainID,
	})

	return tx.Hash().Hex(), nil
}

// Estimate MintAssets cost.
func (api *API) EstimateMintAssets(ctx context.Context, chainID uint64, contractAddress string, walletAddresses []string, amount *bigint.BigInt) (uint64, error) {
	err := api.validateWalletsAndAmounts(walletAddresses, amount)
	if err != nil {
		return 0, err
	}
	usersAddresses, amountsList := api.prepareMintAssetsData(walletAddresses, amount)
	return api.estimateMethod(ctx, chainID, contractAddress, "mintTo", usersAddresses, amountsList)
}

// This is only ERC721 function
func (api *API) RemoteBurn(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, tokenIds []*bigint.BigInt) (string, error) {
	err := api.validateTokens(tokenIds)
	if err != nil {
		return "", err
	}

	contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password))

	var tempTokenIds []*big.Int
	for _, v := range tokenIds {
		tempTokenIds = append(tempTokenIds, v.Int)
	}

	tx, err := contractInst.RemoteBurn(transactOpts, tempTokenIds)
	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(&rpcfilters.PendingTxInfo{
		Hash:    tx.Hash(),
		Type:    string(transactions.RemoteDestructCollectible),
		From:    common.Address(txArgs.From),
		ChainID: chainID,
	})

	return tx.Hash().Hex(), nil
}

// This is only ERC721 function
func (api *API) EstimateRemoteBurn(ctx context.Context, chainID uint64, contractAddress string, tokenIds []*bigint.BigInt) (uint64, error) {
	err := api.validateTokens(tokenIds)
	if err != nil {
		return 0, err
	}

	var tempTokenIds []*big.Int
	for _, v := range tokenIds {
		tempTokenIds = append(tempTokenIds, v.Int)
	}

	return api.estimateMethod(ctx, chainID, contractAddress, "remoteBurn", tempTokenIds)
}

func (api *API) ContractOwner(ctx context.Context, chainID uint64, contractAddress string) (string, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return "", err
	}
	if tokenType == protobuf.CommunityTokenType_ERC721 {
		contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
		if err != nil {
			return "", err
		}
		owner, err := contractInst.Owner(callOpts)
		if err != nil {
			return "", err
		}
		return owner.String(), nil
	} else if tokenType == protobuf.CommunityTokenType_ERC20 {
		contractInst, err := api.newAssetsInstance(chainID, contractAddress)
		if err != nil {
			return "", err
		}
		owner, err := contractInst.Owner(callOpts)
		if err != nil {
			return "", err
		}
		return owner.String(), nil
	}
	return "", fmt.Errorf("unknown token type: %v", tokenType)
}

func (api *API) MintedCount(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	mintedCount, err := contractInst.MintedCount(callOpts)
	if err != nil {
		return nil, err
	}
	return mintedCount, nil
}

func (api *API) RemainingSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
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
	contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
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
	contractInst, err := api.newAssetsInstance(chainID, contractAddress)
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
	contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (api *API) maxSupplyAssets(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.newAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (api *API) maxSupply(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
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

func (api *API) setMaxSupplyCollectibles(transactOpts *bind.TransactOpts, chainID uint64, contractAddress string, newMaxSupply *big.Int) (*types.Transaction, error) {
	contractInst, err := api.newCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.SetMaxSupply(transactOpts, newMaxSupply)
}

func (api *API) setMaxSupplyAssets(transactOpts *bind.TransactOpts, chainID uint64, contractAddress string, newMaxSupply *big.Int) (*types.Transaction, error) {
	contractInst, err := api.newAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.SetMaxSupply(transactOpts, newMaxSupply)
}

func (api *API) setMaxSupply(transactOpts *bind.TransactOpts, chainID uint64, contractAddress string, newMaxSupply *big.Int) (*types.Transaction, error) {
	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return nil, err
	}

	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return api.setMaxSupplyCollectibles(transactOpts, chainID, contractAddress, newMaxSupply)
	case protobuf.CommunityTokenType_ERC20:
		return api.setMaxSupplyAssets(transactOpts, chainID, contractAddress, newMaxSupply)
	default:
		return nil, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (api *API) Burn(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, burnAmount *bigint.BigInt) (string, error) {
	err := api.validateBurnAmount(ctx, burnAmount, chainID, contractAddress)
	if err != nil {
		return "", err
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password))

	newMaxSupply, err := api.prepareNewMaxSupply(ctx, chainID, contractAddress, burnAmount)
	if err != nil {
		return "", err
	}

	tx, err := api.setMaxSupply(transactOpts, chainID, contractAddress, newMaxSupply)
	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(&rpcfilters.PendingTxInfo{
		Hash:    tx.Hash(),
		Type:    string(transactions.BurnCommunityToken),
		From:    common.Address(txArgs.From),
		ChainID: chainID,
	})

	return tx.Hash().Hex(), nil
}

func (api *API) EstimateBurn(ctx context.Context, chainID uint64, contractAddress string, burnAmount *bigint.BigInt) (uint64, error) {
	err := api.validateBurnAmount(ctx, burnAmount, chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	newMaxSupply, err := api.prepareNewMaxSupply(ctx, chainID, contractAddress, burnAmount)
	if err != nil {
		return 0, err
	}

	return api.estimateMethod(ctx, chainID, contractAddress, "setMaxSupply", newMaxSupply)
}

func (api *API) validateWalletsAndAmounts(walletAddresses []string, amount *bigint.BigInt) error {
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

func (api *API) packCollectibleMethod(ctx context.Context, methodName string, args ...interface{}) ([]byte, error) {
	collectiblesABI, err := abi.JSON(strings.NewReader(collectibles.CollectiblesABI))
	if err != nil {
		return []byte{}, err
	}
	return collectiblesABI.Pack(methodName, args...)
}

func (api *API) packAssetsMethod(ctx context.Context, methodName string, args ...interface{}) ([]byte, error) {
	assetsABI, err := abi.JSON(strings.NewReader(assets.AssetsABI))
	if err != nil {
		return []byte{}, err
	}
	return assetsABI.Pack(methodName, args...)
}

func (api *API) estimateMethod(ctx context.Context, chainID uint64, contractAddress string, methodName string, args ...interface{}) (uint64, error) {
	ethClient, err := api.RPCClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return 0, err
	}

	tokenType, err := api.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return 0, err
	}
	var data []byte

	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		data, err = api.packCollectibleMethod(ctx, methodName, args...)
	case protobuf.CommunityTokenType_ERC20:
		data, err = api.packAssetsMethod(ctx, methodName, args...)
	default:
		err = fmt.Errorf("unknown token type: %v", tokenType)
	}
	if err != nil {
		return 0, err
	}

	ownerAddr, err := api.ContractOwner(ctx, chainID, contractAddress)
	if err != nil {
		return 0, err
	}

	toAddr := common.HexToAddress(contractAddress)
	fromAddr := common.HexToAddress(ownerAddr)

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
