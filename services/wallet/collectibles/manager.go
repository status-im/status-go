package collectibles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/circuitbreaker"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/contracts/ierc1155"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const requestTimeout = 5 * time.Second
const signalUpdatedCollectiblesDataPageSize = 10

const EventCollectiblesConnectionStatusChanged walletevent.EventType = "wallet-collectible-status-changed"

// ERC721 does not support function "TokenURI" if call
// returns error starting with one of these strings
var noTokenURIErrorPrefixes = []string{
	"execution reverted",
	"abi: attempting to unmarshall",
}

var (
	ErrAllProvidersFailedForChainID   = errors.New("all providers failed for chainID")
	ErrNoProvidersAvailableForChainID = errors.New("no providers available for chainID")
)

type ManagerInterface interface {
	FetchAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID, asyncFetch bool) ([]thirdparty.FullCollectibleData, error)
	FetchCollectionSocialsAsync(contractID thirdparty.ContractID) error
}

type Manager struct {
	rpcClient rpc.ClientInterface
	providers thirdparty.CollectibleProviders

	httpClient *http.Client

	collectiblesDataDB CollectibleDataStorage
	collectionsDataDB  CollectionDataStorage
	communityManager   *community.Manager
	ownershipDB        *OwnershipDB

	mediaServer *server.MediaServer

	statuses       *sync.Map
	statusNotifier *connection.StatusNotifier
	feed           *event.Feed
	circuitBreaker *circuitbreaker.CircuitBreaker
}

func NewManager(
	db *sql.DB,
	rpcClient rpc.ClientInterface,
	communityManager *community.Manager,
	providers thirdparty.CollectibleProviders,
	mediaServer *server.MediaServer,
	feed *event.Feed) *Manager {

	var ownershipDB *OwnershipDB
	var statuses *sync.Map
	var statusNotifier *connection.StatusNotifier
	if db != nil {
		ownershipDB = NewOwnershipDB(db)
		statuses = initStatuses(ownershipDB)
		statusNotifier = createStatusNotifier(statuses, feed)
	}

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Timeout:                10000,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 25,
		SleepWindow:            300000,
		ErrorPercentThreshold:  25,
	})

	return &Manager{
		rpcClient: rpcClient,
		providers: providers,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		collectiblesDataDB: NewCollectibleDataDB(db),
		collectionsDataDB:  NewCollectionDataDB(db),
		communityManager:   communityManager,
		ownershipDB:        ownershipDB,
		mediaServer:        mediaServer,
		statuses:           statuses,
		statusNotifier:     statusNotifier,
		feed:               feed,
		circuitBreaker:     cb,
	}
}

func mapToList[K comparable, T any](m map[K]T) []T {
	list := make([]T, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}
	return list
}

func (o *Manager) doContentTypeRequest(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logutils.ZapLogger().Error("failed to close head request body", zap.Error(err))
		}
	}()

	return resp.Header.Get("Content-Type"), nil
}

func (o *Manager) getTokenBalancesByOwnerAddress(collectibles *thirdparty.CollectibleContractOwnership, ownerAddress common.Address) map[common.Address][]thirdparty.TokenBalance {
	ret := make(map[common.Address][]thirdparty.TokenBalance)

	for _, nftOwner := range collectibles.Owners {
		if nftOwner.OwnerAddress == ownerAddress {
			ret[collectibles.ContractAddress] = nftOwner.TokenBalances
			break
		}
	}

	return ret
}

func (o *Manager) FetchCachedBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, ownerAddress common.Address, contractAddresses []common.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	ret := make(map[common.Address][]thirdparty.TokenBalance)

	for _, contractAddress := range contractAddresses {
		ret[contractAddress] = make([]thirdparty.TokenBalance, 0)
	}

	for _, contractAddress := range contractAddresses {
		ownership, err := o.ownershipDB.FetchCachedCollectibleOwnersByContractAddress(chainID, contractAddress)
		if err != nil {
			return nil, err
		}

		t := o.getTokenBalancesByOwnerAddress(ownership, ownerAddress)

		for address, tokenBalances := range t {
			ret[address] = append(ret[address], tokenBalances...)
		}
	}

	return ret, nil
}

// Need to combine different providers to support all needed ChainIDs
func (o *Manager) FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, ownerAddress common.Address, contractAddresses []common.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	ret := make(thirdparty.TokenBalancesPerContractAddress)

	for _, contractAddress := range contractAddresses {
		ret[contractAddress] = make([]thirdparty.TokenBalance, 0)
	}

	// Try with account ownership providers first
	assetsContainer, err := o.FetchAllAssetsByOwnerAndContractAddress(ctx, chainID, ownerAddress, contractAddresses, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit, thirdparty.FetchFromAnyProvider)
	if err == ErrNoProvidersAvailableForChainID {
		// Use contract ownership providers
		for _, contractAddress := range contractAddresses {
			ownership, err := o.FetchCollectibleOwnersByContractAddress(ctx, chainID, contractAddress)
			if err != nil {
				return nil, err
			}

			ret = o.getTokenBalancesByOwnerAddress(ownership, ownerAddress)
		}
	} else if err == nil {
		// Account ownership providers succeeded
		for _, fullData := range assetsContainer.Items {
			contractAddress := fullData.CollectibleData.ID.ContractID.Address
			balance := thirdparty.TokenBalance{
				TokenID: fullData.CollectibleData.ID.TokenID,
				Balance: &bigint.BigInt{Int: big.NewInt(1)},
			}
			ret[contractAddress] = append(ret[contractAddress], balance)
		}
	} else {
		// OpenSea could have provided, but returned error
		return nil, err
	}

	return ret, nil
}

func (o *Manager) FetchAllAssetsByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int, providerID string) (*thirdparty.FullCollectibleDataContainer, error) {
	defer o.checkConnectionStatus(chainID)

	cmd := circuitbreaker.NewCommand(ctx, nil)
	for _, provider := range o.providers.AccountOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}
		if providerID != thirdparty.FetchFromAnyProvider && providerID != provider.ID() {
			continue
		}

		provider := provider
		f := circuitbreaker.NewFunctor(
			func() ([]interface{}, error) {
				assetContainer, err := provider.FetchAllAssetsByOwnerAndContractAddress(ctx, chainID, owner, contractAddresses, cursor, limit)
				if err != nil {
					logutils.ZapLogger().Error("FetchAllAssetsByOwnerAndContractAddress failed for",
						zap.String("provider", provider.ID()),
						zap.Stringer("chainID", chainID),
						zap.Error(err))
				}
				return []interface{}{assetContainer}, err
			}, getCircuitName(provider, chainID),
		)
		cmd.Add(f)
	}

	if cmd.IsEmpty() {
		return nil, ErrNoProvidersAvailableForChainID
	}

	cmdRes := o.circuitBreaker.Execute(cmd)
	if cmdRes.Error() != nil {
		logutils.ZapLogger().Error("FetchAllAssetsByOwnerAndContractAddress failed for",
			zap.Stringer("chainID", chainID),
			zap.Error(cmdRes.Error()),
		)
		return nil, cmdRes.Error()
	}

	assetContainer := cmdRes.Result()[0].(*thirdparty.FullCollectibleDataContainer)
	_, err := o.processFullCollectibleData(ctx, assetContainer.Items, true)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchAllAssetsByOwner(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.FullCollectibleDataContainer, error) {
	defer o.checkConnectionStatus(chainID)

	cmd := circuitbreaker.NewCommand(ctx, nil)
	for _, provider := range o.providers.AccountOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}
		if providerID != thirdparty.FetchFromAnyProvider && providerID != provider.ID() {
			continue
		}

		provider := provider
		f := circuitbreaker.NewFunctor(
			func() ([]interface{}, error) {
				assetContainer, err := provider.FetchAllAssetsByOwner(ctx, chainID, owner, cursor, limit)
				if err != nil {
					logutils.ZapLogger().Error("FetchAllAssetsByOwner failed for",
						zap.String("provider", provider.ID()),
						zap.Stringer("chainID", chainID),
						zap.Error(err),
					)
				}
				return []interface{}{assetContainer}, err
			}, getCircuitName(provider, chainID),
		)
		cmd.Add(f)
	}

	if cmd.IsEmpty() {
		return nil, ErrNoProvidersAvailableForChainID
	}

	cmdRes := o.circuitBreaker.Execute(cmd)
	if cmdRes.Error() != nil {
		logutils.ZapLogger().Error("FetchAllAssetsByOwner failed for",
			zap.Stringer("chainID", chainID),
			zap.Error(cmdRes.Error()),
		)
		return nil, cmdRes.Error()
	}

	assetContainer := cmdRes.Result()[0].(*thirdparty.FullCollectibleDataContainer)
	_, err := o.processFullCollectibleData(ctx, assetContainer.Items, true)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchERC1155Balances(ctx context.Context, owner common.Address, chainID walletCommon.ChainID, contractAddress common.Address, tokenIDs []*bigint.BigInt) ([]*bigint.BigInt, error) {
	if len(tokenIDs) == 0 {
		return nil, nil
	}

	backend, err := o.rpcClient.EthClient(uint64(chainID))
	if err != nil {
		return nil, err
	}

	caller, err := ierc1155.NewIerc1155Caller(contractAddress, backend)
	if err != nil {
		return nil, err
	}

	owners := make([]common.Address, len(tokenIDs))
	ids := make([]*big.Int, len(tokenIDs))
	for i, tokenID := range tokenIDs {
		owners[i] = owner
		ids[i] = tokenID.Int
	}

	balances, err := caller.BalanceOfBatch(&bind.CallOpts{
		Context: ctx,
	}, owners, ids)

	if err != nil {
		return nil, err
	}

	bigIntBalances := make([]*bigint.BigInt, len(balances))
	for i, balance := range balances {
		bigIntBalances[i] = &bigint.BigInt{Int: balance}
	}

	return bigIntBalances, err
}

func (o *Manager) fillMissingBalances(ctx context.Context, owner common.Address, collectibles []*thirdparty.FullCollectibleData) {
	collectiblesByChainIDAndContractAddress := thirdparty.GroupCollectiblesByChainIDAndContractAddress(collectibles)

	for chainID, collectiblesByContract := range collectiblesByChainIDAndContractAddress {
		for contractAddress, contractCollectibles := range collectiblesByContract {
			collectiblesToFetchPerTokenID := make(map[string]*thirdparty.FullCollectibleData)

			for _, collectible := range contractCollectibles {
				if collectible.AccountBalance == nil {
					switch getContractType(*collectible) {
					case walletCommon.ContractTypeERC1155:
						collectiblesToFetchPerTokenID[collectible.CollectibleData.ID.TokenID.String()] = collectible
					default:
						// Any other type of collectible is non-fungible, balance is 1
						collectible.AccountBalance = &bigint.BigInt{Int: big.NewInt(1)}
					}
				}
			}

			if len(collectiblesToFetchPerTokenID) == 0 {
				continue
			}

			tokenIDs := make([]*bigint.BigInt, 0, len(collectiblesToFetchPerTokenID))
			for _, c := range collectiblesToFetchPerTokenID {
				tokenIDs = append(tokenIDs, c.CollectibleData.ID.TokenID)
			}

			balances, err := o.FetchERC1155Balances(ctx, owner, chainID, contractAddress, tokenIDs)
			if err != nil {
				logutils.ZapLogger().Error("FetchERC1155Balances failed",
					zap.Stringer("chainID", chainID),
					zap.Stringer("contractAddress", contractAddress),
					zap.Error(err),
				)
				continue
			}

			for i := range balances {
				collectible := collectiblesToFetchPerTokenID[tokenIDs[i].String()]
				collectible.AccountBalance = balances[i]
			}
		}
	}
}

func (o *Manager) FetchCollectibleOwnershipByOwner(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
	// We don't yet have an API that will return only Ownership data
	// Use the full Ownership + Metadata endpoint and use the data we need
	assetContainer, err := o.FetchAllAssetsByOwner(ctx, chainID, owner, cursor, limit, providerID)
	if err != nil {
		return nil, err
	}

	// Some providers do not give us the balances for ERC1155 tokens, so we need to fetch them separately.
	collectibles := make([]*thirdparty.FullCollectibleData, 0, len(assetContainer.Items))
	for i := range assetContainer.Items {
		collectibles = append(collectibles, &assetContainer.Items[i])
	}
	o.fillMissingBalances(ctx, owner, collectibles)

	ret := assetContainer.ToOwnershipContainer()

	return &ret, nil
}

// Returns collectible metadata for the given unique IDs.
// If asyncFetch is true, empty metadata will be returned for any missing collectibles and an EventCollectiblesDataUpdated will be sent when the data is ready.
// If asyncFetch is false, it will wait for all collectibles' metadata to be retrieved before returning.
func (o *Manager) FetchAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID, asyncFetch bool) ([]thirdparty.FullCollectibleData, error) {
	err := o.FetchMissingAssetsByCollectibleUniqueID(ctx, uniqueIDs, asyncFetch)
	if err != nil {
		return nil, err
	}

	return o.getCacheFullCollectibleData(uniqueIDs)
}

func (o *Manager) FetchMissingAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID, asyncFetch bool) error {
	missingIDs, err := o.collectiblesDataDB.GetIDsNotInDB(uniqueIDs)
	if err != nil {
		return err
	}

	missingIDsPerChainID := thirdparty.GroupCollectibleUIDsByChainID(missingIDs)

	// Atomic group stores the error from the first failed command and stops other commands on error
	group := async.NewAtomicGroup(ctx)
	for chainID, idsToFetch := range missingIDsPerChainID {
		group.Add(func(ctx context.Context) error {
			defer o.checkConnectionStatus(chainID)

			fetchedAssets, err := o.fetchMissingAssetsForChainByCollectibleUniqueID(ctx, chainID, idsToFetch)
			if err != nil {
				logutils.ZapLogger().Error("FetchMissingAssetsByCollectibleUniqueID failed for",
					zap.Stringer("chainID", chainID),
					zap.Any("ids", idsToFetch),
					zap.Error(err),
				)
				return err
			}

			updatedCollectibles, err := o.processFullCollectibleData(ctx, fetchedAssets, asyncFetch)
			if err != nil {
				logutils.ZapLogger().Error("processFullCollectibleData failed for",
					zap.Stringer("chainID", chainID),
					zap.Int("len(fetchedAssets)", len(fetchedAssets)),
					zap.Error(err),
				)
				return err
			}

			o.signalUpdatedCollectiblesData(updatedCollectibles)
			return nil
		})
	}

	if asyncFetch {
		group.Wait()
		return group.Error()
	}

	return nil
}

func (o *Manager) fetchMissingAssetsForChainByCollectibleUniqueID(ctx context.Context, chainID walletCommon.ChainID, idsToFetch []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	cmd := circuitbreaker.NewCommand(ctx, nil)
	for _, provider := range o.providers.CollectibleDataProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}

		provider := provider
		cmd.Add(circuitbreaker.NewFunctor(func() ([]any, error) {
			fetchedAssets, err := provider.FetchAssetsByCollectibleUniqueID(ctx, idsToFetch)
			if err != nil {
				logutils.ZapLogger().Error("fetchMissingAssetsForChainByCollectibleUniqueID failed",
					zap.String("provider", provider.ID()),
					zap.Stringer("chainID", chainID),
					zap.Error(err),
				)
			}

			return []any{fetchedAssets}, err
		}, getCircuitName(provider, chainID)))
	}

	if cmd.IsEmpty() {
		return nil, ErrNoProvidersAvailableForChainID // lets not stop the group if no providers are available for the chain
	}

	cmdRes := o.circuitBreaker.Execute(cmd)
	if cmdRes.Error() != nil {
		logutils.ZapLogger().Error("fetchMissingAssetsForChainByCollectibleUniqueID failed for",
			zap.Stringer("chainID", chainID),
			zap.Error(cmdRes.Error()),
		)
		return nil, cmdRes.Error()
	}
	return cmdRes.Result()[0].([]thirdparty.FullCollectibleData), cmdRes.Error()
}

func (o *Manager) FetchCollectionsDataByContractID(ctx context.Context, ids []thirdparty.ContractID) ([]thirdparty.CollectionData, error) {
	missingIDs, err := o.collectionsDataDB.GetIDsNotInDB(ids)
	if err != nil {
		return nil, err
	}

	missingIDsPerChainID := thirdparty.GroupContractIDsByChainID(missingIDs)

	// Atomic group stores the error from the first failed command and stops other commands on error
	group := async.NewAtomicGroup(ctx)
	for chainID, idsToFetch := range missingIDsPerChainID {
		group.Add(func(ctx context.Context) error {
			defer o.checkConnectionStatus(chainID)

			cmd := circuitbreaker.NewCommand(ctx, nil)
			for _, provider := range o.providers.CollectionDataProviders {
				if !provider.IsChainSupported(chainID) {
					continue
				}

				provider := provider
				cmd.Add(circuitbreaker.NewFunctor(func() ([]any, error) {
					fetchedCollections, err := provider.FetchCollectionsDataByContractID(ctx, idsToFetch)
					return []any{fetchedCollections}, err
				}, getCircuitName(provider, chainID)))
			}

			if cmd.IsEmpty() {
				return nil
			}

			cmdRes := o.circuitBreaker.Execute(cmd)
			if cmdRes.Error() != nil {
				logutils.ZapLogger().Error("FetchCollectionsDataByContractID failed for",
					zap.Stringer("chainID", chainID),
					zap.Error(cmdRes.Error()),
				)
				return cmdRes.Error()
			}

			fetchedCollections := cmdRes.Result()[0].([]thirdparty.CollectionData)
			err = o.processCollectionData(ctx, fetchedCollections)
			if err != nil {
				return err
			}

			return err
		})
	}

	group.Wait()

	if group.Error() != nil {
		return nil, group.Error()
	}

	data, err := o.collectionsDataDB.GetData(ids)
	if err != nil {
		return nil, err
	}

	return mapToList(data), nil
}

func (o *Manager) GetCollectibleOwnership(id thirdparty.CollectibleUniqueID) ([]thirdparty.AccountBalance, error) {
	return o.ownershipDB.GetOwnership(id)
}

func (o *Manager) FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	defer o.checkConnectionStatus(chainID)

	cmd := circuitbreaker.NewCommand(ctx, nil)
	for _, provider := range o.providers.ContractOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}

		provider := provider
		cmd.Add(circuitbreaker.NewFunctor(func() ([]any, error) {
			res, err := provider.FetchCollectibleOwnersByContractAddress(ctx, chainID, contractAddress)
			if err != nil {
				logutils.ZapLogger().Error("FetchCollectibleOwnersByContractAddress failed",
					zap.String("provider", provider.ID()),
					zap.Stringer("chainID", chainID),
					zap.Error(err),
				)
			}
			return []any{res}, err
		}, getCircuitName(provider, chainID)))
	}

	if cmd.IsEmpty() {
		return nil, ErrNoProvidersAvailableForChainID
	}

	cmdRes := o.circuitBreaker.Execute(cmd)
	if cmdRes.Error() != nil {
		logutils.ZapLogger().Error("FetchCollectibleOwnersByContractAddress failed for",
			zap.Stringer("chainID", chainID),
			zap.Error(cmdRes.Error()),
		)
		return nil, cmdRes.Error()
	}
	return cmdRes.Result()[0].(*thirdparty.CollectibleContractOwnership), cmdRes.Error()
}

func (o *Manager) fetchTokenURI(ctx context.Context, id thirdparty.CollectibleUniqueID) (string, error) {
	if id.TokenID == nil {
		return "", errors.New("empty token ID")
	}

	backend, err := o.rpcClient.EthClient(uint64(id.ContractID.ChainID))
	if err != nil {
		return "", err
	}

	backend = getClientWithNoCircuitTripping(backend)
	caller, err := collectibles.NewCollectiblesCaller(id.ContractID.Address, backend)

	if err != nil {
		return "", err
	}

	tokenURI, err := caller.TokenURI(&bind.CallOpts{
		Context: ctx,
	}, id.TokenID.Int)

	if err != nil {
		for _, errorPrefix := range noTokenURIErrorPrefixes {
			if strings.Contains(err.Error(), errorPrefix) {
				// Contract doesn't support "TokenURI" method
				return "", nil
			}
		}
		return "", err
	}

	return tokenURI, err
}

func isMetadataEmpty(asset thirdparty.CollectibleData) bool {
	return asset.Description == "" &&
		asset.ImageURL == ""
}

// Processes collectible metadata obtained from a provider and ensures any missing data is fetched.
// If asyncFetch is true, community collectibles metadata will be fetched async and an EventCollectiblesDataUpdated will be sent when the data is ready.
// If asyncFetch is false, it will wait for all community collectibles' metadata to be retrieved before returning.
func (o *Manager) processFullCollectibleData(ctx context.Context, assets []thirdparty.FullCollectibleData, asyncFetch bool) ([]thirdparty.CollectibleUniqueID, error) {
	fullyFetchedAssets := make(map[string]*thirdparty.FullCollectibleData)
	communityCollectibles := make(map[string][]*thirdparty.FullCollectibleData)
	processedIDs := make([]thirdparty.CollectibleUniqueID, 0, len(assets))

	// Start with all assets, remove if any of the fetch steps fail
	for idx := range assets {
		asset := &assets[idx]
		id := asset.CollectibleData.ID
		fullyFetchedAssets[id.HashKey()] = asset
	}

	// Detect community collectibles
	for _, asset := range fullyFetchedAssets {
		// Only check community ownership if metadata is empty
		if isMetadataEmpty(asset.CollectibleData) {
			// Get TokenURI if not given by provider
			err := o.fillTokenURI(ctx, asset)
			if err != nil {
				logutils.ZapLogger().Error("fillTokenURI failed", zap.Error(err))
				delete(fullyFetchedAssets, asset.CollectibleData.ID.HashKey())
				continue
			}

			// Get CommunityID if obtainable from TokenURI
			err = o.fillCommunityID(asset)
			if err != nil {
				logutils.ZapLogger().Error("fillCommunityID failed", zap.Error(err))
				delete(fullyFetchedAssets, asset.CollectibleData.ID.HashKey())
				continue
			}

			// Get metadata from community if community collectible
			communityID := asset.CollectibleData.CommunityID
			if communityID != "" {
				if _, ok := communityCollectibles[communityID]; !ok {
					communityCollectibles[communityID] = make([]*thirdparty.FullCollectibleData, 0)
				}
				communityCollectibles[communityID] = append(communityCollectibles[communityID], asset)

				// Community collectibles are handled separately, remove from list
				delete(fullyFetchedAssets, asset.CollectibleData.ID.HashKey())
			}
		}
	}

	// Community collectibles are grouped by community ID
	for communityID, communityAssets := range communityCollectibles {
		if asyncFetch {
			o.fetchCommunityAssetsAsync(ctx, communityID, communityAssets)
		} else {
			err := o.fetchCommunityAssets(communityID, communityAssets)
			if err != nil {
				logutils.ZapLogger().Error("fetchCommunityAssets failed", zap.String("communityID", communityID), zap.Error(err))
				continue
			}
			for _, asset := range communityAssets {
				processedIDs = append(processedIDs, asset.CollectibleData.ID)
			}
		}
	}

	for _, asset := range fullyFetchedAssets {
		err := o.fillAnimationMediatype(ctx, asset)
		if err != nil {
			logutils.ZapLogger().Error("fillAnimationMediatype failed", zap.Error(err))
			delete(fullyFetchedAssets, asset.CollectibleData.ID.HashKey())
			continue
		}
	}

	// Save successfully fetched data to DB
	collectiblesData := make([]thirdparty.CollectibleData, 0, len(assets))
	collectionsData := make([]thirdparty.CollectionData, 0, len(assets))
	missingCollectionIDs := make([]thirdparty.ContractID, 0)

	for _, asset := range fullyFetchedAssets {
		id := asset.CollectibleData.ID
		processedIDs = append(processedIDs, id)

		collectiblesData = append(collectiblesData, asset.CollectibleData)
		if asset.CollectionData != nil {
			collectionsData = append(collectionsData, *asset.CollectionData)
		} else {
			missingCollectionIDs = append(missingCollectionIDs, id.ContractID)
		}
	}

	err := o.collectiblesDataDB.SetData(collectiblesData, true)
	if err != nil {
		return nil, err
	}

	err = o.collectionsDataDB.SetData(collectionsData, true)
	if err != nil {
		return nil, err
	}

	if len(missingCollectionIDs) > 0 {
		// Calling this ensures collection data is fetched and cached (if not already available)
		_, err := o.FetchCollectionsDataByContractID(ctx, missingCollectionIDs)
		if err != nil {
			return nil, err
		}
	}

	return processedIDs, nil
}

func (o *Manager) fillTokenURI(ctx context.Context, asset *thirdparty.FullCollectibleData) error {
	id := asset.CollectibleData.ID

	tokenURI := asset.CollectibleData.TokenURI
	// Only need to fetch it from contract if it was empty
	if tokenURI == "" {
		tokenURI, err := o.fetchTokenURI(ctx, id)

		if err != nil {
			return err
		}

		asset.CollectibleData.TokenURI = tokenURI
	}
	return nil
}

func (o *Manager) fillCommunityID(asset *thirdparty.FullCollectibleData) error {
	tokenURI := asset.CollectibleData.TokenURI

	communityID := ""
	if tokenURI != "" {
		communityID = o.communityManager.GetCommunityID(tokenURI)
	}

	asset.CollectibleData.CommunityID = communityID
	return nil
}

func (o *Manager) fetchCommunityAssets(communityID string, communityAssets []*thirdparty.FullCollectibleData) error {
	communityFound, err := o.communityManager.FillCollectiblesMetadata(communityID, communityAssets)
	if err != nil {
		logutils.ZapLogger().Error("FillCollectiblesMetadata failed", zap.String("communityID", communityID), zap.Error(err))
	} else if !communityFound {
		logutils.ZapLogger().Warn("fetchCommunityAssets community not found", zap.String("communityID", communityID))
	}

	// If the community is found, we update the DB.
	// If the community is not found, we only insert new entries to the DB (don't replace what is already there).
	allowUpdate := communityFound

	collectiblesData := make([]thirdparty.CollectibleData, 0, len(communityAssets))
	collectionsData := make([]thirdparty.CollectionData, 0, len(communityAssets))

	for _, asset := range communityAssets {
		collectiblesData = append(collectiblesData, asset.CollectibleData)
		if asset.CollectionData != nil {
			collectionsData = append(collectionsData, *asset.CollectionData)
		}
	}

	err = o.collectiblesDataDB.SetData(collectiblesData, allowUpdate)
	if err != nil {
		logutils.ZapLogger().Error("collectiblesDataDB SetData failed", zap.String("communityID", communityID), zap.Error(err))
		return err
	}

	err = o.collectionsDataDB.SetData(collectionsData, allowUpdate)
	if err != nil {
		logutils.ZapLogger().Error("collectionsDataDB SetData failed", zap.String("communityID", communityID), zap.Error(err))
		return err
	}

	for _, asset := range communityAssets {
		if asset.CollectibleCommunityInfo != nil {
			err = o.collectiblesDataDB.SetCommunityInfo(asset.CollectibleData.ID, *asset.CollectibleCommunityInfo)
			if err != nil {
				logutils.ZapLogger().Error("collectiblesDataDB SetCommunityInfo failed", zap.String("communityID", communityID), zap.Error(err))
				return err
			}
		}
	}

	return nil
}

func (o *Manager) fetchCommunityAssetsAsync(_ context.Context, communityID string, communityAssets []*thirdparty.FullCollectibleData) {
	if len(communityAssets) == 0 {
		return
	}

	go func() {
		defer gocommon.LogOnPanic()
		err := o.fetchCommunityAssets(communityID, communityAssets)
		if err != nil {
			logutils.ZapLogger().Error("fetchCommunityAssets failed", zap.String("communityID", communityID), zap.Error(err))
			return
		}

		// Metadata is up to date in db at this point, fetch and send Event.
		ids := make([]thirdparty.CollectibleUniqueID, 0, len(communityAssets))
		for _, asset := range communityAssets {
			ids = append(ids, asset.CollectibleData.ID)
		}
		o.signalUpdatedCollectiblesData(ids)
	}()
}

func (o *Manager) fillAnimationMediatype(ctx context.Context, asset *thirdparty.FullCollectibleData) error {
	if len(asset.CollectibleData.AnimationURL) > 0 {
		contentType, err := o.doContentTypeRequest(ctx, asset.CollectibleData.AnimationURL)
		if err != nil {
			asset.CollectibleData.AnimationURL = ""
		}
		asset.CollectibleData.AnimationMediaType = contentType
	}
	return nil
}

func (o *Manager) processCollectionData(_ context.Context, collections []thirdparty.CollectionData) error {
	return o.collectionsDataDB.SetData(collections, true)
}

func (o *Manager) getCacheFullCollectibleData(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	ret := make([]thirdparty.FullCollectibleData, 0, len(uniqueIDs))

	collectiblesData, err := o.collectiblesDataDB.GetData(uniqueIDs)
	if err != nil {
		return nil, err
	}

	contractIDs := make([]thirdparty.ContractID, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		contractIDs = append(contractIDs, id.ContractID)
	}

	collectionsData, err := o.collectionsDataDB.GetData(contractIDs)
	if err != nil {
		return nil, err
	}

	for _, id := range uniqueIDs {
		collectibleData, ok := collectiblesData[id.HashKey()]
		if !ok {
			// Use empty data, set only ID
			collectibleData = thirdparty.CollectibleData{
				ID: id,
			}
		}
		if o.mediaServer != nil && len(collectibleData.ImagePayload) > 0 {
			collectibleData.ImageURL = o.mediaServer.MakeWalletCollectibleImagesURL(collectibleData.ID)
		}

		collectionData, ok := collectionsData[id.ContractID.HashKey()]
		if !ok {
			// Use empty data, set only ID
			collectionData = thirdparty.CollectionData{
				ID: id.ContractID,
			}
		}
		if o.mediaServer != nil && len(collectionData.ImagePayload) > 0 {
			collectionData.ImageURL = o.mediaServer.MakeWalletCollectionImagesURL(collectionData.ID)
		}

		communityInfo, _, err := o.communityManager.GetCommunityInfo(collectibleData.CommunityID)
		if err != nil {
			return nil, err
		}

		collectibleCommunityInfo, err := o.collectiblesDataDB.GetCommunityInfo(id)
		if err != nil {
			return nil, err
		}

		ownership, err := o.ownershipDB.GetOwnership(id)
		if err != nil {
			return nil, err
		}

		fullData := thirdparty.FullCollectibleData{
			CollectibleData:          collectibleData,
			CollectionData:           &collectionData,
			CommunityInfo:            communityInfo,
			CollectibleCommunityInfo: collectibleCommunityInfo,
			Ownership:                ownership,
		}
		ret = append(ret, fullData)
	}

	return ret, nil
}

func (o *Manager) SetCollectibleTransferID(ownerAddress common.Address, id thirdparty.CollectibleUniqueID, transferID common.Hash, notify bool) error {
	changed, err := o.ownershipDB.SetTransferID(ownerAddress, id, transferID)
	if err != nil {
		return err
	}

	if changed && notify {
		o.signalUpdatedCollectiblesData([]thirdparty.CollectibleUniqueID{id})
	}
	return nil
}

// Reset connection status to trigger notifications
// on the next status update
func (o *Manager) ResetConnectionStatus() {
	o.statuses.Range(func(key, value interface{}) bool {
		value.(*connection.Status).ResetStateValue()
		return true
	})
}

func (o *Manager) checkConnectionStatus(chainID walletCommon.ChainID) {
	for _, provider := range o.providers.GetProviderList() {
		if provider.IsChainSupported(chainID) && provider.IsConnected() {
			if status, ok := o.statuses.Load(chainID.String()); ok {
				status.(*connection.Status).SetIsConnected(true)
			}
			return
		}
	}

	// If no chain in statuses, add it
	statusVal, ok := o.statuses.Load(chainID.String())
	if !ok {
		status := connection.NewStatus()
		status.SetIsConnected(false)
		o.statuses.Store(chainID.String(), status)
		o.updateStatusNotifier()
	} else {
		statusVal.(*connection.Status).SetIsConnected(false)
	}
}

func (o *Manager) signalUpdatedCollectiblesData(ids []thirdparty.CollectibleUniqueID) {
	// We limit how much collectibles data we send in each event to avoid problems on the client side
	for startIdx := 0; startIdx < len(ids); startIdx += signalUpdatedCollectiblesDataPageSize {
		endIdx := startIdx + signalUpdatedCollectiblesDataPageSize
		if endIdx > len(ids) {
			endIdx = len(ids)
		}
		pageIDs := ids[startIdx:endIdx]

		collectibles, err := o.getCacheFullCollectibleData(pageIDs)
		if err != nil {
			logutils.ZapLogger().Error("Error getting FullCollectibleData from cache", zap.Error(err))
			return
		}

		// Send update event with most complete data type available
		details := fullCollectiblesDataToDetails(collectibles)

		payload, err := json.Marshal(details)
		if err != nil {
			logutils.ZapLogger().Error("Error marshaling response", zap.Error(err))
			return
		}

		event := walletevent.Event{
			Type:    EventCollectiblesDataUpdated,
			Message: string(payload),
		}

		o.feed.Send(event)
	}
}

func (o *Manager) SearchCollectibles(ctx context.Context, chainID walletCommon.ChainID, text string, cursor string, limit int, providerID string) (*thirdparty.FullCollectibleDataContainer, error) {
	defer o.checkConnectionStatus(chainID)

	anyProviderAvailable := false
	for _, provider := range o.providers.SearchProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}
		anyProviderAvailable = true
		if providerID != thirdparty.FetchFromAnyProvider && providerID != provider.ID() {
			continue
		}

		// TODO (#13951): Be smarter about how we handle the user-entered string
		collections := []common.Address{}

		container, err := provider.SearchCollectibles(ctx, chainID, collections, text, cursor, limit)
		if err != nil {
			logutils.ZapLogger().Error("FetchAllAssetsByOwner failed for",
				zap.String("provider", provider.ID()),
				zap.Stringer("chainID", chainID),
				zap.Error(err),
			)
			continue
		}

		_, err = o.processFullCollectibleData(ctx, container.Items, true)
		if err != nil {
			return nil, err
		}

		return container, nil
	}

	if anyProviderAvailable {
		return nil, ErrAllProvidersFailedForChainID
	}
	return nil, ErrNoProvidersAvailableForChainID
}

func (o *Manager) SearchCollections(ctx context.Context, chainID walletCommon.ChainID, query string, cursor string, limit int, providerID string) (*thirdparty.CollectionDataContainer, error) {
	defer o.checkConnectionStatus(chainID)

	anyProviderAvailable := false
	for _, provider := range o.providers.SearchProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}
		anyProviderAvailable = true
		if providerID != thirdparty.FetchFromAnyProvider && providerID != provider.ID() {
			continue
		}

		// TODO (#13951): Be smarter about how we handle the user-entered string
		container, err := provider.SearchCollections(ctx, chainID, query, cursor, limit)
		if err != nil {
			logutils.ZapLogger().Error("FetchAllAssetsByOwner failed for",
				zap.String("provider", provider.ID()),
				zap.Stringer("chainID", chainID),
				zap.Error(err),
			)
			continue
		}

		err = o.processCollectionData(ctx, container.Items)
		if err != nil {
			return nil, err
		}

		return container, nil
	}

	if anyProviderAvailable {
		return nil, ErrAllProvidersFailedForChainID
	}
	return nil, ErrNoProvidersAvailableForChainID
}

func (o *Manager) FetchCollectionSocialsAsync(contractID thirdparty.ContractID) error {
	go func() {
		defer gocommon.LogOnPanic()
		defer o.checkConnectionStatus(contractID.ChainID)

		socials, err := o.getOrFetchSocialsForCollection(context.Background(), contractID)
		if err != nil || socials == nil {
			logutils.ZapLogger().Debug("FetchCollectionSocialsAsync failed for",
				zap.Stringer("chainID", contractID.ChainID),
				zap.Stringer("address", contractID.Address),
				zap.Error(err),
			)
			return
		}

		socialsMessage := CollectionSocialsMessage{
			ID:      contractID,
			Socials: socials,
		}

		payload, err := json.Marshal(socialsMessage)
		if err != nil {
			logutils.ZapLogger().Error("Error marshaling response", zap.Error(err))
			return
		}

		event := walletevent.Event{
			Type:    EventGetCollectionSocialsDone,
			Message: string(payload),
		}

		o.feed.Send(event)
	}()

	return nil
}

func (o *Manager) getOrFetchSocialsForCollection(_ context.Context, contractID thirdparty.ContractID) (*thirdparty.CollectionSocials, error) {
	socials, err := o.collectionsDataDB.GetSocialsForID(contractID)
	if err != nil {
		logutils.ZapLogger().Debug("getOrFetchSocialsForCollection failed for",
			zap.Stringer("chainID", contractID.ChainID),
			zap.Stringer("address", contractID.Address),
			zap.Error(err),
		)
		return nil, err
	}
	if socials == nil {
		return o.fetchSocialsForCollection(context.Background(), contractID)
	}
	return socials, nil
}

func (o *Manager) fetchSocialsForCollection(ctx context.Context, contractID thirdparty.ContractID) (*thirdparty.CollectionSocials, error) {
	cmd := circuitbreaker.NewCommand(ctx, nil)
	for _, provider := range o.providers.CollectibleDataProviders {
		if !provider.IsChainSupported(contractID.ChainID) {
			continue
		}

		provider := provider
		cmd.Add(circuitbreaker.NewFunctor(func() ([]interface{}, error) {
			socials, err := provider.FetchCollectionSocials(ctx, contractID)
			if err != nil {
				logutils.ZapLogger().Error("FetchCollectionSocials failed for",
					zap.String("provider", provider.ID()),
					zap.Stringer("chainID", contractID.ChainID),
					zap.Error(err),
				)
			}
			return []interface{}{socials}, err
		}, getCircuitName(provider, contractID.ChainID)))
	}

	if cmd.IsEmpty() {
		return nil, ErrNoProvidersAvailableForChainID // lets not stop the group if no providers are available for the chain
	}

	cmdRes := o.circuitBreaker.Execute(cmd)
	if cmdRes.Error() != nil {
		logutils.ZapLogger().Error("fetchSocialsForCollection failed for",
			zap.Stringer("chainID", contractID.ChainID),
			zap.Error(cmdRes.Error()),
		)
		return nil, cmdRes.Error()
	}

	socials := cmdRes.Result()[0].(*thirdparty.CollectionSocials)
	err := o.collectionsDataDB.SetCollectionSocialsData(contractID, socials)
	if err != nil {
		logutils.ZapLogger().Error("Error saving socials to DB", zap.Error(err))
		return nil, err
	}

	return socials, cmdRes.Error()
}

func (o *Manager) updateStatusNotifier() {
	o.statusNotifier = createStatusNotifier(o.statuses, o.feed)
}

func initStatuses(ownershipDB *OwnershipDB) *sync.Map {
	statuses := &sync.Map{}
	for _, chainID := range walletCommon.AllChainIDs() {
		status := connection.NewStatus()
		state := status.GetState()
		latestUpdateTimestamp, err := ownershipDB.GetLatestOwnershipUpdateTimestamp(chainID)
		if err == nil {
			state.LastSuccessAt = latestUpdateTimestamp
			status.SetState(state)
		}
		statuses.Store(chainID.String(), status)
	}

	return statuses
}

func createStatusNotifier(statuses *sync.Map, feed *event.Feed) *connection.StatusNotifier {
	return connection.NewStatusNotifier(
		statuses,
		EventCollectiblesConnectionStatusChanged,
		feed,
	)
}

// Different providers have API keys per chain or per testnet/mainnet.
// Proper implementation should respect that. For now, the safest solution is to use the provider ID and chain ID as the key.
func getCircuitName(provider thirdparty.CollectibleProvider, chainID walletCommon.ChainID) string {
	return provider.ID() + chainID.String()
}

func getCircuitNameForTokenURI(mainCircuitName string) string {
	return mainCircuitName + "_tokenURI"
}

// As we don't use hystrix internal way of switching to another circuit, just its metrics,
// we still can switch to another provider without tripping the circuit.
func getClientWithNoCircuitTripping(backend chain.ClientInterface) chain.ClientInterface {
	copyable := backend.(chain.Copyable)
	if copyable != nil {
		backendCopy := copyable.Copy().(chain.ClientInterface)
		hm := backendCopy.(chain.HealthMonitor)
		if hm != nil {
			cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
				Timeout:               20000,
				MaxConcurrentRequests: 100,
				SleepWindow:           300000,
				ErrorPercentThreshold: 101, // Always healthy
			})
			cb.SetOverrideCircuitNameHandler(func(circuitName string) string {
				return getCircuitNameForTokenURI(circuitName)
			})
			hm.SetCircuitBreaker(cb)
			backend = backendCopy
		}
	}

	return backend
}
