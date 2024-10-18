package communitytokens

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/community-tokens/mastertoken"
	"github.com/status-im/status-go/contracts/community-tokens/ownertoken"
	communityownertokenregistry "github.com/status-im/status-go/contracts/community-tokens/registry"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/communitytokens/communitytokensdatabase"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wallet/bigint"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

// Collectibles service
type Service struct {
	manager         *Manager
	accountsManager *account.GethManager
	pendingTracker  *transactions.PendingTxTracker
	config          *params.NodeConfig
	db              *communitytokensdatabase.Database
	Messenger       *protocol.Messenger
	walletFeed      *event.Feed
	walletWatcher   *walletevent.Watcher
	transactor      *transactions.Transactor
	feeManager      *fees.FeeManager
}

// Returns a new Collectibles Service.
func NewService(rpcClient *rpc.Client, accountsManager *account.GethManager, pendingTracker *transactions.PendingTxTracker,
	config *params.NodeConfig, appDb *sql.DB, walletFeed *event.Feed, transactor *transactions.Transactor) *Service {
	return &Service{
		manager:         &Manager{rpcClient: rpcClient},
		accountsManager: accountsManager,
		pendingTracker:  pendingTracker,
		config:          config,
		db:              communitytokensdatabase.NewCommunityTokensDatabase(appDb),
		walletFeed:      walletFeed,
		transactor:      transactor,
		feeManager:      &fees.FeeManager{RPCClient: rpcClient},
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "communitytokens",
			Version:   "0.1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
func (s *Service) Start() error {

	s.walletWatcher = walletevent.NewWatcher(s.walletFeed, s.handleWalletEvent)
	s.walletWatcher.Start()

	return nil
}

func (s *Service) handleWalletEvent(event walletevent.Event) {
	if event.Type == transactions.EventPendingTransactionStatusChanged {
		var p transactions.StatusChangedPayload
		err := json.Unmarshal([]byte(event.Message), &p)
		if err != nil {
			logutils.ZapLogger().Error(errors.Wrap(err, fmt.Sprintf("can't parse transaction message %v\n", event.Message)).Error())
			return
		}
		if p.Status == transactions.Pending {
			return
		}
		pendingTransaction, err := s.pendingTracker.GetPendingEntry(p.ChainID, p.Hash)
		if err != nil {
			logutils.ZapLogger().Error(errors.Wrap(err, fmt.Sprintf("no pending transaction with hash %v on chain %v\n", p.Hash, p.ChainID)).Error())
			return
		}

		var communityToken, ownerToken, masterToken *token.CommunityToken = &token.CommunityToken{}, &token.CommunityToken{}, &token.CommunityToken{}
		var tokenErr error
		switch pendingTransaction.Type {
		case transactions.DeployCommunityToken:
			communityToken, tokenErr = s.handleDeployCommunityToken(p.Status, pendingTransaction)
		case transactions.AirdropCommunityToken:
			communityToken, tokenErr = s.handleAirdropCommunityToken(p.Status, pendingTransaction)
		case transactions.RemoteDestructCollectible:
			communityToken, tokenErr = s.handleRemoteDestructCollectible(p.Status, pendingTransaction)
		case transactions.BurnCommunityToken:
			communityToken, tokenErr = s.handleBurnCommunityToken(p.Status, pendingTransaction)
		case transactions.DeployOwnerToken:
			ownerToken, masterToken, tokenErr = s.handleDeployOwnerToken(p.Status, pendingTransaction)
		case transactions.SetSignerPublicKey:
			communityToken, tokenErr = s.handleSetSignerPubKey(p.Status, pendingTransaction)
		default:
			return
		}

		err = s.pendingTracker.Delete(context.Background(), p.ChainID, p.Hash)
		if err != nil {
			logutils.ZapLogger().Error(errors.Wrap(err, fmt.Sprintf("can't delete pending transaction with hash %v on chain %v\n", p.Hash, p.ChainID)).Error())
		}

		errorStr := ""
		if tokenErr != nil {
			errorStr = tokenErr.Error()
		}

		signal.SendCommunityTokenTransactionStatusSignal(string(pendingTransaction.Type), p.Status == transactions.Success, pendingTransaction.Hash,
			communityToken, ownerToken, masterToken, errorStr)
	}
}

func (s *Service) handleAirdropCommunityToken(status string, pendingTransaction *transactions.PendingTransaction) (*token.CommunityToken, error) {
	communityToken, err := s.Messenger.GetCommunityTokenByChainAndAddress(int(pendingTransaction.ChainID), pendingTransaction.To.String())
	if communityToken == nil {
		return nil, fmt.Errorf("token does not exist in database: chainId=%v, address=%v", pendingTransaction.ChainID, pendingTransaction.To.String())
	} else {
		publishErr := s.publishTokenActionToPrivilegedMembers(communityToken.CommunityID, uint64(communityToken.ChainID),
			communityToken.Address, protobuf.CommunityTokenAction_AIRDROP)
		if publishErr != nil {
			logutils.ZapLogger().Warn("can't publish airdrop action")
		}
	}
	return communityToken, err
}

func (s *Service) handleRemoteDestructCollectible(status string, pendingTransaction *transactions.PendingTransaction) (*token.CommunityToken, error) {
	communityToken, err := s.Messenger.GetCommunityTokenByChainAndAddress(int(pendingTransaction.ChainID), pendingTransaction.To.String())
	if communityToken == nil {
		return nil, fmt.Errorf("token does not exist in database: chainId=%v, address=%v", pendingTransaction.ChainID, pendingTransaction.To.String())
	} else {
		publishErr := s.publishTokenActionToPrivilegedMembers(communityToken.CommunityID, uint64(communityToken.ChainID),
			communityToken.Address, protobuf.CommunityTokenAction_REMOTE_DESTRUCT)
		if publishErr != nil {
			logutils.ZapLogger().Warn("can't publish remote destruct action")
		}
	}
	return communityToken, err
}

func (s *Service) handleBurnCommunityToken(status string, pendingTransaction *transactions.PendingTransaction) (*token.CommunityToken, error) {
	if status == transactions.Success {
		// get new max supply and update database
		newMaxSupply, err := s.maxSupply(context.Background(), uint64(pendingTransaction.ChainID), pendingTransaction.To.String())
		if err != nil {
			return nil, err
		}
		err = s.Messenger.UpdateCommunityTokenSupply(int(pendingTransaction.ChainID), pendingTransaction.To.String(), &bigint.BigInt{Int: newMaxSupply})
		if err != nil {
			return nil, err
		}
	}

	communityToken, err := s.Messenger.GetCommunityTokenByChainAndAddress(int(pendingTransaction.ChainID), pendingTransaction.To.String())

	if communityToken == nil {
		return nil, fmt.Errorf("token does not exist in database: chainId=%v, address=%v", pendingTransaction.ChainID, pendingTransaction.To.String())
	} else {
		publishErr := s.publishTokenActionToPrivilegedMembers(communityToken.CommunityID, uint64(communityToken.ChainID),
			communityToken.Address, protobuf.CommunityTokenAction_BURN)
		if publishErr != nil {
			logutils.ZapLogger().Warn("can't publish burn action")
		}
	}
	return communityToken, err
}

func (s *Service) handleDeployOwnerToken(status string, pendingTransaction *transactions.PendingTransaction) (*token.CommunityToken, *token.CommunityToken, error) {
	newMasterAddress, err := s.GetMasterTokenContractAddressFromHash(context.Background(), uint64(pendingTransaction.ChainID), pendingTransaction.Hash.Hex())
	if err != nil {
		return nil, nil, err
	}
	newOwnerAddress, err := s.GetOwnerTokenContractAddressFromHash(context.Background(), uint64(pendingTransaction.ChainID), pendingTransaction.Hash.Hex())
	if err != nil {
		return nil, nil, err
	}

	err = s.Messenger.UpdateCommunityTokenAddress(int(pendingTransaction.ChainID), s.TemporaryOwnerContractAddress(pendingTransaction.Hash.Hex()), newOwnerAddress)
	if err != nil {
		return nil, nil, err
	}
	err = s.Messenger.UpdateCommunityTokenAddress(int(pendingTransaction.ChainID), s.TemporaryMasterContractAddress(pendingTransaction.Hash.Hex()), newMasterAddress)
	if err != nil {
		return nil, nil, err
	}

	ownerToken, err := s.updateStateAndAddTokenToCommunityDescription(status, int(pendingTransaction.ChainID), newOwnerAddress)
	if err != nil {
		return nil, nil, err
	}

	masterToken, err := s.updateStateAndAddTokenToCommunityDescription(status, int(pendingTransaction.ChainID), newMasterAddress)
	if err != nil {
		return nil, nil, err
	}

	return ownerToken, masterToken, nil
}

func (s *Service) updateStateAndAddTokenToCommunityDescription(status string, chainID int, address string) (*token.CommunityToken, error) {
	tokenToUpdate, err := s.Messenger.GetCommunityTokenByChainAndAddress(chainID, address)
	if err != nil {
		return nil, err
	}
	if tokenToUpdate == nil {
		return nil, fmt.Errorf("token does not exist in database: chainID=%v, address=%v", chainID, address)
	}

	if status == transactions.Success {
		err := s.Messenger.UpdateCommunityTokenState(chainID, address, token.Deployed)
		if err != nil {
			return nil, err
		}
		err = s.Messenger.AddCommunityToken(tokenToUpdate.CommunityID, chainID, address)
		if err != nil {
			return nil, err
		}
	} else {
		err := s.Messenger.UpdateCommunityTokenState(chainID, address, token.Failed)
		if err != nil {
			return nil, err
		}
	}
	return s.Messenger.GetCommunityTokenByChainAndAddress(chainID, address)
}

func (s *Service) handleDeployCommunityToken(status string, pendingTransaction *transactions.PendingTransaction) (*token.CommunityToken, error) {
	return s.updateStateAndAddTokenToCommunityDescription(status, int(pendingTransaction.ChainID), pendingTransaction.To.String())
}

func (s *Service) handleSetSignerPubKey(status string, pendingTransaction *transactions.PendingTransaction) (*token.CommunityToken, error) {

	communityToken, err := s.Messenger.GetCommunityTokenByChainAndAddress(int(pendingTransaction.ChainID), pendingTransaction.To.String())
	if err != nil {
		return nil, err
	}
	if communityToken == nil {
		return nil, fmt.Errorf("token does not exist in database: chainId=%v, address=%v", pendingTransaction.ChainID, pendingTransaction.To.String())
	}

	if status == transactions.Success {
		_, err := s.Messenger.PromoteSelfToControlNode(types.FromHex(communityToken.CommunityID))
		if err != nil {
			return nil, err
		}
	}
	return communityToken, err
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	s.walletWatcher.Stop()
	return nil
}

func (s *Service) Init(messenger *protocol.Messenger) {
	s.Messenger = messenger
}

func (s *Service) NewCommunityOwnerTokenRegistryInstance(chainID uint64, contractAddress string) (*communityownertokenregistry.CommunityOwnerTokenRegistry, error) {
	backend, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return communityownertokenregistry.NewCommunityOwnerTokenRegistry(common.HexToAddress(contractAddress), backend)
}

func (s *Service) NewOwnerTokenInstance(chainID uint64, contractAddress string) (*ownertoken.OwnerToken, error) {

	backend, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return ownertoken.NewOwnerToken(common.HexToAddress(contractAddress), backend)

}

func (s *Service) NewMasterTokenInstance(chainID uint64, contractAddress string) (*mastertoken.MasterToken, error) {
	backend, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return mastertoken.NewMasterToken(common.HexToAddress(contractAddress), backend)
}

func (s *Service) validateTokens(tokenIds []*bigint.BigInt) error {
	if len(tokenIds) == 0 {
		return errors.New("token list is empty")
	}
	return nil
}

func (s *Service) validateBurnAmount(ctx context.Context, burnAmount *bigint.BigInt, chainID uint64, contractAddress string) error {
	if burnAmount.Cmp(big.NewInt(0)) <= 0 {
		return errors.New("burnAmount is less than 0")
	}
	remainingSupply, err := s.remainingSupply(ctx, chainID, contractAddress)
	if err != nil {
		return err
	}
	if burnAmount.Cmp(remainingSupply.Int) > 1 {
		return errors.New("burnAmount is bigger than remaining amount")
	}
	return nil
}

func (s *Service) remainingSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	tokenType, err := s.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return s.remainingCollectiblesSupply(ctx, chainID, contractAddress)
	case protobuf.CommunityTokenType_ERC20:
		return s.remainingAssetsSupply(ctx, chainID, contractAddress)
	default:
		return nil, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (s *Service) prepareNewMaxSupply(ctx context.Context, chainID uint64, contractAddress string, burnAmount *bigint.BigInt) (*big.Int, error) {
	maxSupply, err := s.maxSupply(ctx, chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	var newMaxSupply = new(big.Int)
	newMaxSupply.Sub(maxSupply, burnAmount.Int)
	return newMaxSupply, nil
}

// RemainingSupply = MaxSupply - MintedCount
func (s *Service) remainingCollectiblesSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.manager.NewCollectiblesInstance(chainID, contractAddress)
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
func (s *Service) remainingAssetsSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.manager.NewAssetsInstance(chainID, contractAddress)
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

func (s *Service) ValidateWalletsAndAmounts(walletAddresses []string, amount *bigint.BigInt) error {
	if len(walletAddresses) == 0 {
		return errors.New("wallet addresses list is empty")
	}
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return errors.New("amount is <= 0")
	}
	return nil
}

func (s *Service) GetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string) (string, error) {

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.NewOwnerTokenInstance(chainID, contractAddress)
	if err != nil {
		return "", err
	}
	signerPubKey, err := contractInst.SignerPublicKey(callOpts)
	if err != nil {
		return "", err
	}

	return types.ToHex(signerPubKey), nil
}

func (s *Service) SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error) {
	// 1. Get Owner Token contract address from deployer contract - SafeGetOwnerTokenAddress()
	ownerTokenAddr, err := s.SafeGetOwnerTokenAddress(ctx, chainID, communityID)
	if err != nil {
		return "", err
	}
	// 2. Get Signer from owner token contract - GetSignerPubKey()
	return s.GetSignerPubKey(ctx, chainID, ownerTokenAddr)
}

func (s *Service) SafeGetOwnerTokenAddress(ctx context.Context, chainID uint64, communityID string) (string, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	deployerContractInst, err := s.manager.NewCommunityTokenDeployerInstance(chainID)
	if err != nil {
		return "", err
	}
	registryAddr, err := deployerContractInst.DeploymentRegistry(callOpts)
	if err != nil {
		return "", err
	}
	registryContractInst, err := s.NewCommunityOwnerTokenRegistryInstance(chainID, registryAddr.Hex())
	if err != nil {
		return "", err
	}
	communityEthAddress, err := convert33BytesPubKeyToEthAddress(communityID)
	if err != nil {
		return "", err
	}
	ownerTokenAddress, err := registryContractInst.GetEntry(callOpts, communityEthAddress)

	return ownerTokenAddress.Hex(), err
}

func (s *Service) GetCollectibleContractData(chainID uint64, contractAddress string) (*communities.CollectibleContractData, error) {
	return s.manager.GetCollectibleContractData(chainID, contractAddress)
}

func (s *Service) GetAssetContractData(chainID uint64, contractAddress string) (*communities.AssetContractData, error) {
	return s.manager.GetAssetContractData(chainID, contractAddress)
}

func (s *Service) DeploymentSignatureDigest(chainID uint64, addressFrom string, communityID string) ([]byte, error) {
	return s.manager.DeploymentSignatureDigest(chainID, addressFrom, communityID)
}

func (s *Service) ProcessCommunityTokenAction(message *protobuf.CommunityTokenAction) error {
	communityToken, err := s.Messenger.GetCommunityTokenByChainAndAddress(int(message.ChainId), message.ContractAddress)
	if err != nil {
		return err
	}
	if communityToken == nil {
		return fmt.Errorf("can't find community token in database: chain %v, address %v", message.ChainId, message.ContractAddress)
	}

	if message.ActionType == protobuf.CommunityTokenAction_BURN {
		// get new max supply and update database
		newMaxSupply, err := s.maxSupply(context.Background(), uint64(communityToken.ChainID), communityToken.Address)
		if err != nil {
			return nil
		}
		err = s.Messenger.UpdateCommunityTokenSupply(communityToken.ChainID, communityToken.Address, &bigint.BigInt{Int: newMaxSupply})
		if err != nil {
			return err
		}
		communityToken, _ = s.Messenger.GetCommunityTokenByChainAndAddress(int(message.ChainId), message.ContractAddress)
	}

	signal.SendCommunityTokenActionSignal(communityToken, message.ActionType)

	return nil
}

func (s *Service) SetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, newSignerPubKey string) (string, error) {

	if len(newSignerPubKey) <= 0 {
		return "", fmt.Errorf("signerPubKey is empty")
	}

	transactOpts := txArgs.ToTransactOpts(utils.GetSigner(chainID, s.accountsManager, s.config.KeyStoreDir, txArgs.From, password))

	contractInst, err := s.NewOwnerTokenInstance(chainID, contractAddress)
	if err != nil {
		return "", err
	}

	tx, err := contractInst.SetSignerPublicKey(transactOpts, common.FromHex(newSignerPubKey))
	if err != nil {
		return "", err
	}

	err = s.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		common.HexToAddress(contractAddress),
		transactions.SetSignerPublicKey,
		transactions.Keep,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func (s *Service) maxSupplyCollectibles(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.manager.NewCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (s *Service) maxSupplyAssets(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := s.manager.NewAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst.MaxSupply(callOpts)
}

func (s *Service) maxSupply(ctx context.Context, chainID uint64, contractAddress string) (*big.Int, error) {
	tokenType, err := s.db.GetTokenType(chainID, contractAddress)
	if err != nil {
		return nil, err
	}

	switch tokenType {
	case protobuf.CommunityTokenType_ERC721:
		return s.maxSupplyCollectibles(ctx, chainID, contractAddress)
	case protobuf.CommunityTokenType_ERC20:
		return s.maxSupplyAssets(ctx, chainID, contractAddress)
	default:
		return nil, fmt.Errorf("unknown token type: %v", tokenType)
	}
}

func (s *Service) CreateCommunityTokenAndSave(chainID int, deploymentParameters DeploymentParameters,
	deployerAddress string, contractAddress string, tokenType protobuf.CommunityTokenType, privilegesLevel token.PrivilegesLevel, transactionHash string) (*token.CommunityToken, error) {

	contractVersion := ""
	if privilegesLevel == token.CommunityLevel {
		contractVersion = s.currentVersion()
	}

	tokenToSave := &token.CommunityToken{
		TokenType:          tokenType,
		CommunityID:        deploymentParameters.CommunityID,
		Address:            contractAddress,
		Name:               deploymentParameters.Name,
		Symbol:             deploymentParameters.Symbol,
		Description:        deploymentParameters.Description,
		Supply:             &bigint.BigInt{Int: deploymentParameters.GetSupply()},
		InfiniteSupply:     deploymentParameters.InfiniteSupply,
		Transferable:       deploymentParameters.Transferable,
		RemoteSelfDestruct: deploymentParameters.RemoteSelfDestruct,
		ChainID:            chainID,
		DeployState:        token.InProgress,
		Decimals:           deploymentParameters.Decimals,
		Deployer:           deployerAddress,
		PrivilegesLevel:    privilegesLevel,
		Base64Image:        deploymentParameters.Base64Image,
		TransactionHash:    transactionHash,
		Version:            contractVersion,
	}

	return s.Messenger.SaveCommunityToken(tokenToSave, deploymentParameters.CroppedImage)
}

const (
	MasterSuffix = "-master"
	OwnerSuffix  = "-owner"
)

func (s *Service) TemporaryMasterContractAddress(hash string) string {
	return hash + MasterSuffix
}

func (s *Service) TemporaryOwnerContractAddress(hash string) string {
	return hash + OwnerSuffix
}

func (s *Service) HashFromTemporaryContractAddress(address string) string {
	if strings.HasSuffix(address, OwnerSuffix) {
		return strings.TrimSuffix(address, OwnerSuffix)
	} else if strings.HasSuffix(address, MasterSuffix) {
		return strings.TrimSuffix(address, MasterSuffix)
	}
	return ""
}

func (s *Service) GetMasterTokenContractAddressFromHash(ctx context.Context, chainID uint64, txHash string) (string, error) {
	ethClient, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return "", err
	}

	receipt, err := ethClient.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return "", err
	}

	deployerContractInst, err := s.manager.NewCommunityTokenDeployerInstance(chainID)
	if err != nil {
		return "", err
	}

	logMasterTokenCreatedSig := []byte("DeployMasterToken(address)")
	logMasterTokenCreatedSigHash := crypto.Keccak256Hash(logMasterTokenCreatedSig)

	for _, vLog := range receipt.Logs {
		if vLog.Topics[0].Hex() == logMasterTokenCreatedSigHash.Hex() {
			event, err := deployerContractInst.ParseDeployMasterToken(*vLog)
			if err != nil {
				return "", err
			}
			return event.Arg0.Hex(), nil
		}
	}
	return "", fmt.Errorf("can't find master token address in transaction: %v", txHash)
}

func (s *Service) GetOwnerTokenContractAddressFromHash(ctx context.Context, chainID uint64, txHash string) (string, error) {
	ethClient, err := s.manager.rpcClient.EthClient(chainID)
	if err != nil {
		return "", err
	}

	receipt, err := ethClient.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return "", err
	}

	deployerContractInst, err := s.manager.NewCommunityTokenDeployerInstance(chainID)
	if err != nil {
		return "", err
	}

	logOwnerTokenCreatedSig := []byte("DeployOwnerToken(address)")
	logOwnerTokenCreatedSigHash := crypto.Keccak256Hash(logOwnerTokenCreatedSig)

	for _, vLog := range receipt.Logs {
		if vLog.Topics[0].Hex() == logOwnerTokenCreatedSigHash.Hex() {
			event, err := deployerContractInst.ParseDeployOwnerToken(*vLog)
			if err != nil {
				return "", err
			}
			return event.Arg0.Hex(), nil
		}
	}
	return "", fmt.Errorf("can't find owner token address in transaction: %v", txHash)
}

func (s *Service) ReTrackOwnerTokenDeploymentTransaction(ctx context.Context, chainID uint64, contractAddress string) error {
	communityToken, err := s.Messenger.GetCommunityTokenByChainAndAddress(int(chainID), contractAddress)
	if err != nil {
		return err
	}
	if communityToken == nil {
		return fmt.Errorf("can't find token with address %v on chain %v", contractAddress, chainID)
	}
	if communityToken.DeployState != token.InProgress {
		return fmt.Errorf("token with address %v on chain %v is not in progress", contractAddress, chainID)
	}

	hashString := communityToken.TransactionHash
	if hashString == "" && (communityToken.PrivilegesLevel == token.OwnerLevel || communityToken.PrivilegesLevel == token.MasterLevel) {
		hashString = s.HashFromTemporaryContractAddress(communityToken.Address)
	}

	if hashString == "" {
		return fmt.Errorf("can't find transaction hash for token with address %v on chain %v", contractAddress, chainID)
	}

	transactionType := transactions.DeployCommunityToken
	if communityToken.PrivilegesLevel == token.OwnerLevel || communityToken.PrivilegesLevel == token.MasterLevel {
		transactionType = transactions.DeployOwnerToken
	}

	_, err = s.pendingTracker.GetPendingEntry(wcommon.ChainID(chainID), common.HexToHash(hashString))
	if errors.Is(err, sql.ErrNoRows) {
		// start only if no pending transaction in database
		err = s.pendingTracker.TrackPendingTransaction(
			wcommon.ChainID(chainID),
			common.HexToHash(hashString),
			common.HexToAddress(communityToken.Deployer),
			common.Address{},
			transactionType,
			transactions.Keep,
			"",
		)
		logutils.ZapLogger().Debug("retracking pending transaction", zap.String("hashId", hashString))
	} else {
		logutils.ZapLogger().Debug("pending transaction already tracked", zap.String("hashId", hashString))
	}
	return err
}

func (s *Service) publishTokenActionToPrivilegedMembers(communityID string, chainID uint64, contractAddress string, actionType protobuf.CommunityTokenAction_ActionType) error {
	decodedCommunityID, err := types.DecodeHex(communityID)
	if err != nil {
		return err
	}
	return s.Messenger.PublishTokenActionToPrivilegedMembers(decodedCommunityID, chainID, contractAddress, actionType)
}
