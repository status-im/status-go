package collectibles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const requestTimeout = 5 * time.Second
const failedCommunityFetchRetryDelay = 1 * time.Hour

const hystrixContractOwnershipClientName = "contractOwnershipClient"

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
	FetchAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error)
}

type Manager struct {
	rpcClient                  *rpc.Client
	contractOwnershipProviders []thirdparty.CollectibleContractOwnershipProvider
	accountOwnershipProviders  []thirdparty.CollectibleAccountOwnershipProvider
	collectibleDataProviders   []thirdparty.CollectibleDataProvider
	collectionDataProviders    []thirdparty.CollectionDataProvider
	collectibleProviders       []thirdparty.CollectibleProvider
	communityInfoProvider      thirdparty.CollectibleCommunityInfoProvider

	httpClient *http.Client

	collectiblesDataDB *CollectibleDataDB
	collectionsDataDB  *CollectionDataDB
	communityDataDB    *community.DataDB

	statuses       map[string]*connection.Status
	statusNotifier *connection.StatusNotifier
}

func NewManager(
	db *sql.DB,
	rpcClient *rpc.Client,
	contractOwnershipProviders []thirdparty.CollectibleContractOwnershipProvider,
	accountOwnershipProviders []thirdparty.CollectibleAccountOwnershipProvider,
	collectibleDataProviders []thirdparty.CollectibleDataProvider,
	collectionDataProviders []thirdparty.CollectionDataProvider,
	feed *event.Feed) *Manager {
	hystrix.ConfigureCommand(hystrixContractOwnershipClientName, hystrix.CommandConfig{
		Timeout:               10000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	ownershipDB := NewOwnershipDB(db)

	statuses := make(map[string]*connection.Status)

	allChainIDs := walletCommon.AllChainIDs()
	for _, chainID := range allChainIDs {
		status := connection.NewStatus()
		state := status.GetState()
		latestUpdateTimestamp, err := ownershipDB.GetLatestOwnershipUpdateTimestamp(chainID)
		if err == nil {
			state.LastSuccessAt = latestUpdateTimestamp
			status.SetState(state)
		}
		statuses[chainID.String()] = status
	}

	statusNotifier := connection.NewStatusNotifier(
		statuses,
		EventCollectiblesConnectionStatusChanged,
		feed,
	)

	// Get list of all providers
	collectibleProvidersMap := make(map[string]thirdparty.CollectibleProvider)
	collectibleProviders := make([]thirdparty.CollectibleProvider, 0)
	for _, provider := range contractOwnershipProviders {
		collectibleProvidersMap[provider.ID()] = provider
	}
	for _, provider := range accountOwnershipProviders {
		collectibleProvidersMap[provider.ID()] = provider
	}
	for _, provider := range collectibleDataProviders {
		collectibleProvidersMap[provider.ID()] = provider
	}
	for _, provider := range collectionDataProviders {
		collectibleProvidersMap[provider.ID()] = provider
	}
	for _, provider := range collectibleProvidersMap {
		collectibleProviders = append(collectibleProviders, provider)
	}

	return &Manager{
		rpcClient:                  rpcClient,
		contractOwnershipProviders: contractOwnershipProviders,
		accountOwnershipProviders:  accountOwnershipProviders,
		collectibleDataProviders:   collectibleDataProviders,
		collectionDataProviders:    collectionDataProviders,
		collectibleProviders:       collectibleProviders,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		collectiblesDataDB: NewCollectibleDataDB(db),
		collectionsDataDB:  NewCollectionDataDB(db),
		communityDataDB:    community.NewDataDB(db),
		statuses:           statuses,
		statusNotifier:     statusNotifier,
	}
}

func mapToList[K comparable, T any](m map[K]T) []T {
	list := make([]T, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}
	return list
}

func makeContractOwnershipCall(main func() (any, error), fallback func() (any, error)) (any, error) {
	resultChan := make(chan any, 1)
	errChan := hystrix.Go(hystrixContractOwnershipClientName, func() error {
		res, err := main()
		if err != nil {
			return err
		}
		resultChan <- res
		return nil
	}, func(err error) error {
		if fallback == nil {
			return err
		}

		res, err := fallback()
		if err != nil {
			return err
		}
		resultChan <- res
		return nil
	})
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	}
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
			log.Error("failed to close head request body", "err", err)
		}
	}()

	return resp.Header.Get("Content-Type"), nil
}

// Used to break circular dependency, call once as soon as possible after initialization
func (o *Manager) SetCommunityInfoProvider(communityInfoProvider thirdparty.CollectibleCommunityInfoProvider) {
	o.communityInfoProvider = communityInfoProvider
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
			for _, nftOwner := range ownership.Owners {
				if nftOwner.OwnerAddress == ownerAddress {
					ret[contractAddress] = nftOwner.TokenBalances
					break
				}
			}
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

	anyProviderAvailable := false
	for _, provider := range o.accountOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}
		anyProviderAvailable = true
		if providerID != thirdparty.FetchFromAnyProvider && providerID != provider.ID() {
			continue
		}

		assetContainer, err := provider.FetchAllAssetsByOwnerAndContractAddress(ctx, chainID, owner, contractAddresses, cursor, limit)
		if err != nil {
			log.Error("FetchAllAssetsByOwnerAndContractAddress failed for", "provider", provider.ID(), "chainID", chainID, "err", err)
			continue
		}

		err = o.processFullCollectibleData(ctx, assetContainer.Items)
		if err != nil {
			return nil, err
		}

		return assetContainer, nil
	}

	if anyProviderAvailable {
		return nil, ErrAllProvidersFailedForChainID
	}
	return nil, ErrNoProvidersAvailableForChainID
}

func (o *Manager) FetchAllAssetsByOwner(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.FullCollectibleDataContainer, error) {
	defer o.checkConnectionStatus(chainID)

	anyProviderAvailable := false
	for _, provider := range o.accountOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}
		anyProviderAvailable = true
		if providerID != thirdparty.FetchFromAnyProvider && providerID != provider.ID() {
			continue
		}

		assetContainer, err := provider.FetchAllAssetsByOwner(ctx, chainID, owner, cursor, limit)
		if err != nil {
			log.Error("FetchAllAssetsByOwner failed for", "provider", provider.ID(), "chainID", chainID, "err", err)
			continue
		}

		err = o.processFullCollectibleData(ctx, assetContainer.Items)
		if err != nil {
			return nil, err
		}

		return assetContainer, nil
	}

	if anyProviderAvailable {
		return nil, ErrAllProvidersFailedForChainID
	}
	return nil, ErrNoProvidersAvailableForChainID
}

func (o *Manager) FetchCollectibleOwnershipByOwner(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
	// We don't yet have an API that will return only Ownership data
	// Use the full Ownership + Metadata endpoint and use the data we need
	assetContainer, err := o.FetchAllAssetsByOwner(ctx, chainID, owner, cursor, limit, providerID)
	if err != nil {
		return nil, err
	}

	ret := assetContainer.ToOwnershipContainer()
	return &ret, nil
}

func (o *Manager) FetchAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	missingIDs, err := o.collectiblesDataDB.GetIDsNotInDB(uniqueIDs)
	if err != nil {
		return nil, err
	}

	missingIDsPerChainID := thirdparty.GroupCollectibleUIDsByChainID(missingIDs)

	for chainID, idsToFetch := range missingIDsPerChainID {
		defer o.checkConnectionStatus(chainID)

		for _, provider := range o.collectibleDataProviders {
			if !provider.IsChainSupported(chainID) {
				continue
			}

			fetchedAssets, err := provider.FetchAssetsByCollectibleUniqueID(ctx, idsToFetch)
			if err != nil {
				log.Error("FetchAssetsByCollectibleUniqueID failed for", "provider", provider.ID(), "chainID", chainID, "err", err)
				continue
			}

			err = o.processFullCollectibleData(ctx, fetchedAssets)
			if err != nil {
				return nil, err
			}

			break
		}
	}

	return o.getCacheFullCollectibleData(uniqueIDs)
}

func (o *Manager) FetchCollectionsDataByContractID(ctx context.Context, ids []thirdparty.ContractID) ([]thirdparty.CollectionData, error) {
	missingIDs, err := o.collectionsDataDB.GetIDsNotInDB(ids)
	if err != nil {
		return nil, err
	}

	missingIDsPerChainID := thirdparty.GroupContractIDsByChainID(missingIDs)

	for chainID, idsToFetch := range missingIDsPerChainID {
		defer o.checkConnectionStatus(chainID)

		for _, provider := range o.collectionDataProviders {
			if !provider.IsChainSupported(chainID) {
				continue
			}

			fetchedCollections, err := provider.FetchCollectionsDataByContractID(ctx, idsToFetch)
			if err != nil {
				log.Error("FetchCollectionsDataByContractID failed for", "provider", provider.ID(), "chainID", chainID, "err", err)
				continue
			}

			err = o.processCollectionData(ctx, fetchedCollections)
			if err != nil {
				return nil, err
			}

			break
		}
	}

	data, err := o.collectionsDataDB.GetData(ids)
	if err != nil {
		return nil, err
	}

	return mapToList(data), nil
}

func (o *Manager) getContractOwnershipProviders(chainID walletCommon.ChainID) (mainProvider thirdparty.CollectibleContractOwnershipProvider, fallbackProvider thirdparty.CollectibleContractOwnershipProvider) {
	mainProvider = nil
	fallbackProvider = nil

	for _, provider := range o.contractOwnershipProviders {
		if provider.IsChainSupported(chainID) {
			if mainProvider == nil {
				// First provider found
				mainProvider = provider
				continue
			}
			// Second provider found
			fallbackProvider = provider
			break
		}
	}
	return
}

func getCollectibleOwnersByContractAddressFunc(ctx context.Context, chainID walletCommon.ChainID, contractAddress common.Address, provider thirdparty.CollectibleContractOwnershipProvider) func() (any, error) {
	if provider == nil {
		return nil
	}
	return func() (any, error) {
		res, err := provider.FetchCollectibleOwnersByContractAddress(ctx, chainID, contractAddress)
		if err != nil {
			log.Error("FetchCollectibleOwnersByContractAddress failed for", "provider", provider.ID(), "chainID", chainID, "err", err)
		}
		return res, err
	}
}

func (o *Manager) FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	defer o.checkConnectionStatus(chainID)

	mainProvider, fallbackProvider := o.getContractOwnershipProviders(chainID)
	if mainProvider == nil {
		return nil, ErrNoProvidersAvailableForChainID
	}

	mainFn := getCollectibleOwnersByContractAddressFunc(ctx, chainID, contractAddress, mainProvider)
	fallbackFn := getCollectibleOwnersByContractAddressFunc(ctx, chainID, contractAddress, fallbackProvider)

	owners, err := makeContractOwnershipCall(mainFn, fallbackFn)
	if err != nil {
		return nil, err
	}

	return owners.(*thirdparty.CollectibleContractOwnership), nil
}

func isMetadataEmpty(asset thirdparty.CollectibleData) bool {
	return asset.Name == "" &&
		asset.Description == "" &&
		asset.ImageURL == ""
}

func (o *Manager) fetchTokenURI(ctx context.Context, id thirdparty.CollectibleUniqueID) (string, error) {
	if id.TokenID == nil {
		return "", errors.New("empty token ID")
	}
	backend, err := o.rpcClient.EthClient(uint64(id.ContractID.ChainID))
	if err != nil {
		return "", err
	}

	caller, err := collectibles.NewCollectiblesCaller(id.ContractID.Address, backend)
	if err != nil {
		return "", err
	}

	tokenURI, err := caller.TokenURI(&bind.CallOpts{
		Context: ctx,
	}, id.TokenID.Int)

	if err != nil {
		for _, errorPrefix := range noTokenURIErrorPrefixes {
			if strings.HasPrefix(err.Error(), errorPrefix) {
				// Contract doesn't support "TokenURI" method
				return "", nil
			}
		}
		return "", err
	}

	return tokenURI, err
}

func (o *Manager) processFullCollectibleData(ctx context.Context, assets []thirdparty.FullCollectibleData) error {
	fullyFetchedAssets := make(map[string]*thirdparty.FullCollectibleData)
	communityCollectibles := make(map[string][]*thirdparty.FullCollectibleData)

	// Start with all assets, remove if any of the fetch steps fail
	for idx := range assets {
		asset := &assets[idx]
		id := asset.CollectibleData.ID
		fullyFetchedAssets[id.HashKey()] = asset
	}

	for _, asset := range fullyFetchedAssets {
		// Only check community ownership if metadata is empty
		if isMetadataEmpty(asset.CollectibleData) {
			err := o.fillTokenURI(ctx, asset)
			if err != nil {
				log.Error("fillTokenURI failed", "err", err)
				delete(fullyFetchedAssets, asset.CollectibleData.ID.HashKey())
				continue
			}
			err = o.fillCommunityID(asset)
			if err != nil {
				log.Error("fillCommunityID failed", "err", err)
				delete(fullyFetchedAssets, asset.CollectibleData.ID.HashKey())
				continue
			}

			communityID := asset.CollectibleData.CommunityID
			if communityID != "" {
				if _, ok := communityCollectibles[communityID]; !ok {
					communityCollectibles[communityID] = make([]*thirdparty.FullCollectibleData, 0)
				}
				communityCollectibles[communityID] = append(communityCollectibles[communityID], asset)
			}
		}
	}

	// Community collectibles are grouped by community ID
	// If fetching data for one community fails (for example, owner node is down),
	// skip and continue with the other communities.
	for communityID, communityAssets := range communityCollectibles {
		err := o.fillCommunityInfo(communityID, communityAssets)
		if err != nil {
			log.Error("fillCommunityInfo failed", "communityID", communityID, "err", err)
			for _, communityAsset := range communityAssets {
				delete(fullyFetchedAssets, communityAsset.CollectibleData.ID.HashKey())
			}
		}
	}

	for _, asset := range fullyFetchedAssets {
		err := o.fillAnimationMediatype(ctx, asset)
		if err != nil {
			log.Error("fillAnimationMediatype failed", "err", err)
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

		collectiblesData = append(collectiblesData, asset.CollectibleData)
		if asset.CollectionData != nil {
			collectionsData = append(collectionsData, *asset.CollectionData)
		} else {
			missingCollectionIDs = append(missingCollectionIDs, id.ContractID)
		}
	}

	err := o.collectiblesDataDB.SetData(collectiblesData)
	if err != nil {
		return err
	}

	for _, asset := range assets {
		if asset.CommunityInfo != nil {
			err = o.collectiblesDataDB.SetCommunityInfo(asset.CollectibleData.ID, *asset.CommunityInfo)
			if err != nil {
				return err
			}
		}
	}

	err = o.collectionsDataDB.SetData(collectionsData)
	if err != nil {
		return err
	}

	if len(missingCollectionIDs) > 0 {
		// Calling this ensures collection data is fetched and cached (if not already available)
		_, err := o.FetchCollectionsDataByContractID(ctx, missingCollectionIDs)
		if err != nil {
			return err
		}
	}

	return nil
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
		communityID = o.communityInfoProvider.GetCommunityID(tokenURI)
	}

	asset.CollectibleData.CommunityID = communityID
	return nil
}

func (o *Manager) mustFetchCommunityInfo(communityID string) bool {
	// See if we have cached data
	_, state, err := o.communityDataDB.GetCommunityInfo(communityID)
	if err != nil {
		return true
	}

	// If we don't have a state, this community has never been fetched before
	if state == nil {
		return true
	}

	// If the last fetch was successful, we can safely refresh our cache
	if state.LastUpdateSuccesful {
		return true
	}

	// If the last fetch was not successful, we should only retry after a delay
	if time.Unix(int64(state.LastUpdateTimestamp), 0).Add(failedCommunityFetchRetryDelay).Before(time.Now()) {
		return true
	}

	return false
}

func (o *Manager) fetchCommunityInfo(communityID string) (*thirdparty.CommunityInfo, error) {
	if !o.mustFetchCommunityInfo(communityID) {
		return nil, fmt.Errorf("backing off fetchCommunityInfo for id: %s", communityID)
	}

	communityInfo, err := o.communityInfoProvider.FetchCommunityInfo(communityID)
	if err != nil {
		dbErr := o.communityDataDB.SetCommunityInfo(communityID, nil)
		if dbErr != nil {
			log.Error("SetCommunityInfo failed", "communityID", communityID, "err", dbErr)
		}
		return nil, err
	}

	err = o.communityDataDB.SetCommunityInfo(communityID, communityInfo)
	return communityInfo, err
}

func (o *Manager) fillCommunityInfo(communityID string, communityAssets []*thirdparty.FullCollectibleData) error {
	communityInfo, err := o.fetchCommunityInfo(communityID)
	if err != nil {
		return err
	}

	if communityInfo != nil {
		for _, communityAsset := range communityAssets {
			err := o.communityInfoProvider.FillCollectibleMetadata(communityAsset)
			if err != nil {
				return err
			}
		}
	}

	return nil
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

func (o *Manager) processCollectionData(ctx context.Context, collections []thirdparty.CollectionData) error {
	return o.collectionsDataDB.SetData(collections)
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

		collectionData, ok := collectionsData[id.ContractID.HashKey()]
		if !ok {
			// Use empty data, set only ID
			collectionData = thirdparty.CollectionData{
				ID: id.ContractID,
			}
		}

		communityInfo, err := o.collectiblesDataDB.GetCommunityInfo(id)
		if err != nil {
			return nil, err
		}

		fullData := thirdparty.FullCollectibleData{
			CollectibleData: collectibleData,
			CollectionData:  &collectionData,
			CommunityInfo:   communityInfo,
		}
		ret = append(ret, fullData)
	}

	return ret, nil
}

// Reset connection status to trigger notifications
// on the next status update
func (o *Manager) ResetConnectionStatus() {
	for _, status := range o.statuses {
		status.ResetStateValue()
	}
}

func (o *Manager) checkConnectionStatus(chainID walletCommon.ChainID) {
	for _, provider := range o.collectibleProviders {
		if provider.IsChainSupported(chainID) && provider.IsConnected() {
			o.statuses[chainID.String()].SetIsConnected(true)
			return
		}
	}
	o.statuses[chainID.String()].SetIsConnected(false)
}
