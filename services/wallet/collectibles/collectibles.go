package collectibles

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts/collectibles"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
)

const requestTimeout = 5 * time.Second

const hystrixContractOwnershipClientName = "contractOwnershipClient"

// ERC721 does not support function "TokenURI" if call
// returns error starting with one of these strings
var noTokenURIErrorPrefixes = []string{
	"execution reverted",
	"abi: attempting to unmarshall",
}

type Manager struct {
	rpcClient                         *rpc.Client
	mainContractOwnershipProvider     thirdparty.CollectibleContractOwnershipProvider
	fallbackContractOwnershipProvider thirdparty.CollectibleContractOwnershipProvider
	metadataProvider                  thirdparty.CollectibleMetadataProvider
	opensea                           *opensea.Client
	httpClient                        *http.Client
	collectiblesDataCache             map[string]thirdparty.CollectibleData
	collectiblesDataCacheLock         sync.RWMutex
	collectionsDataCache              map[string]thirdparty.CollectionData
	collectionsDataCacheLock          sync.RWMutex
}

func NewManager(rpcClient *rpc.Client, mainContractOwnershipProvider thirdparty.CollectibleContractOwnershipProvider, fallbackContractOwnershipProvider thirdparty.CollectibleContractOwnershipProvider, opensea *opensea.Client) *Manager {
	hystrix.ConfigureCommand(hystrixContractOwnershipClientName, hystrix.CommandConfig{
		Timeout:               10000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	return &Manager{
		rpcClient:                         rpcClient,
		mainContractOwnershipProvider:     mainContractOwnershipProvider,
		fallbackContractOwnershipProvider: fallbackContractOwnershipProvider,
		opensea:                           opensea,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		collectiblesDataCache: make(map[string]thirdparty.CollectibleData),
		collectionsDataCache:  make(map[string]thirdparty.CollectionData),
	}
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

func (o *Manager) doContentTypeRequest(url string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
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
func (o *Manager) SetMetadataProvider(metadataProvider thirdparty.CollectibleMetadataProvider) {
	o.metadataProvider = metadataProvider
}

func (o *Manager) FetchAllCollectionsByOwner(chainID walletCommon.ChainID, owner common.Address) ([]opensea.OwnedCollection, error) {
	return o.opensea.FetchAllCollectionsByOwner(chainID, owner)
}

func (o *Manager) FetchAllOpenseaAssetsByOwnerAndCollection(chainID walletCommon.ChainID, owner common.Address, collectionSlug string, cursor string, limit int) (*opensea.AssetContainer, error) {
	return o.opensea.FetchAllOpenseaAssetsByOwnerAndCollection(chainID, owner, collectionSlug, cursor, limit)
}

func (o *Manager) FetchAllAssetsByOwnerAndCollection(chainID walletCommon.ChainID, owner common.Address, collectionSlug string, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assetContainer, err := o.opensea.FetchAllAssetsByOwnerAndCollection(chainID, owner, collectionSlug, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processFullCollectibleData(assetContainer.Items)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

// Need to combine different providers to support all needed ChainIDs
func (o *Manager) FetchBalancesByOwnerAndContractAddress(chainID walletCommon.ChainID, ownerAddress common.Address, contractAddresses []common.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	ret := make(thirdparty.TokenBalancesPerContractAddress)

	for _, contractAddress := range contractAddresses {
		ret[contractAddress] = make([]thirdparty.TokenBalance, 0)
	}

	// Try with more direct endpoint first (OpenSea)
	assetsContainer, err := o.FetchAllAssetsByOwnerAndContractAddress(chainID, ownerAddress, contractAddresses, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	if err == thirdparty.ErrChainIDNotSupported {
		// Use contract ownership providers
		for _, contractAddress := range contractAddresses {
			ownership, err := o.FetchCollectibleOwnersByContractAddress(chainID, contractAddress)
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
		// OpenSea could provide
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

func (o *Manager) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assetContainer, err := o.opensea.FetchAllAssetsByOwnerAndContractAddress(chainID, owner, contractAddresses, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processFullCollectibleData(assetContainer.Items)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assetContainer, err := o.opensea.FetchAllAssetsByOwner(chainID, owner, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processFullCollectibleData(assetContainer.Items)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchCollectibleOwnershipByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.CollectibleOwnershipContainer, error) {
	// We don't yet have an API that will return only Ownership data
	// Use the full Ownership + Metadata endpoint and use the data we need
	assetContainer, err := o.FetchAllAssetsByOwner(chainID, owner, cursor, limit)
	if err != nil {
		return nil, err
	}

	ret := assetContainer.ToOwnershipContainer()
	return &ret, nil
}

func (o *Manager) FetchAssetsByCollectibleUniqueID(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	idsToFetch := o.getIDsNotInCollectiblesDataCache(uniqueIDs)
	if len(idsToFetch) > 0 {
		fetchedAssets, err := o.opensea.FetchAssetsByCollectibleUniqueID(idsToFetch)
		if err != nil {
			return nil, err
		}

		err = o.processFullCollectibleData(fetchedAssets)
		if err != nil {
			return nil, err
		}
	}

	return o.getCacheFullCollectibleData(uniqueIDs), nil
}

func (o *Manager) FetchCollectibleOwnersByContractAddress(chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	mainFunc := func() (any, error) {
		return o.mainContractOwnershipProvider.FetchCollectibleOwnersByContractAddress(chainID, contractAddress)
	}
	var fallbackFunc func() (any, error) = nil
	if o.fallbackContractOwnershipProvider != nil && o.fallbackContractOwnershipProvider.IsChainSupported(chainID) {
		fallbackFunc = func() (any, error) {
			return o.fallbackContractOwnershipProvider.FetchCollectibleOwnersByContractAddress(chainID, contractAddress)
		}
	}
	owners, err := makeContractOwnershipCall(mainFunc, fallbackFunc)
	if err != nil {
		return nil, err
	}

	return owners.(*thirdparty.CollectibleContractOwnership), nil
}

func isMetadataEmpty(asset thirdparty.CollectibleData) bool {
	return asset.Name == "" &&
		asset.Description == "" &&
		asset.ImageURL == "" &&
		asset.TokenURI == ""
}

func (o *Manager) fetchTokenURI(id thirdparty.CollectibleUniqueID) (string, error) {
	backend, err := o.rpcClient.EthClient(uint64(id.ContractID.ChainID))
	if err != nil {
		return "", err
	}

	caller, err := collectibles.NewCollectiblesCaller(id.ContractID.Address, backend)
	if err != nil {
		return "", err
	}

	timeoutContext, timeoutCancel := context.WithTimeout(context.Background(), requestTimeout)
	defer timeoutCancel()

	tokenURI, err := caller.TokenURI(&bind.CallOpts{
		Context: timeoutContext,
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

func (o *Manager) processFullCollectibleData(assets []thirdparty.FullCollectibleData) error {
	for idx, asset := range assets {
		id := asset.CollectibleData.ID

		// Get Metadata from alternate source if empty
		if isMetadataEmpty(asset.CollectibleData) {
			if o.metadataProvider == nil {
				return fmt.Errorf("CollectibleMetadataProvider not available")
			}
			tokenURI, err := o.fetchTokenURI(id)

			if err != nil {
				return err
			}

			assets[idx].CollectibleData.TokenURI = tokenURI

			canProvide, err := o.metadataProvider.CanProvideCollectibleMetadata(id, tokenURI)

			if err != nil {
				return err
			}

			if canProvide {
				metadata, err := o.metadataProvider.FetchCollectibleMetadata(id, tokenURI)
				if err != nil {
					return err
				}

				if metadata != nil {
					assets[idx] = *metadata
				}
			}
		}

		// Get Animation MediaType
		if len(assets[idx].CollectibleData.AnimationURL) > 0 {
			contentType, err := o.doContentTypeRequest(assets[idx].CollectibleData.AnimationURL)
			if err != nil {
				assets[idx].CollectibleData.AnimationURL = ""
			}
			assets[idx].CollectibleData.AnimationMediaType = contentType
		}

		o.setCacheCollectibleData(assets[idx].CollectibleData)
		if assets[idx].CollectionData != nil {
			o.setCacheCollectionData(*assets[idx].CollectionData)
		}
	}

	return nil
}

func (o *Manager) isIDInCollectiblesDataCache(id thirdparty.CollectibleUniqueID) bool {
	o.collectiblesDataCacheLock.RLock()
	defer o.collectiblesDataCacheLock.RUnlock()
	if _, ok := o.collectiblesDataCache[id.HashKey()]; ok {
		return true
	}
	return false
}

func (o *Manager) getIDsNotInCollectiblesDataCache(uniqueIDs []thirdparty.CollectibleUniqueID) []thirdparty.CollectibleUniqueID {
	idsToFetch := make([]thirdparty.CollectibleUniqueID, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		if o.isIDInCollectiblesDataCache(id) {
			continue
		}
		idsToFetch = append(idsToFetch, id)
	}
	return idsToFetch
}

func (o *Manager) getCacheCollectiblesData(uniqueIDs []thirdparty.CollectibleUniqueID) map[string]*thirdparty.CollectibleData {
	o.collectiblesDataCacheLock.RLock()
	defer o.collectiblesDataCacheLock.RUnlock()

	collectibles := make(map[string]*thirdparty.CollectibleData)
	for _, id := range uniqueIDs {
		if collectible, ok := o.collectiblesDataCache[id.HashKey()]; ok {
			collectibles[id.HashKey()] = &collectible
			continue
		}
	}
	return collectibles
}

func (o *Manager) setCacheCollectibleData(data thirdparty.CollectibleData) {
	o.collectiblesDataCacheLock.Lock()
	defer o.collectiblesDataCacheLock.Unlock()

	o.collectiblesDataCache[data.ID.HashKey()] = data
}

func (o *Manager) isIDInContractDataCache(id thirdparty.ContractID) bool {
	o.collectionsDataCacheLock.RLock()
	defer o.collectionsDataCacheLock.RUnlock()
	if _, ok := o.collectionsDataCache[id.HashKey()]; ok {
		return true
	}
	return false
}

func (o *Manager) getIDsNotInContractDataCache(ids []thirdparty.ContractID) []thirdparty.ContractID {
	idsToFetch := make([]thirdparty.ContractID, 0, len(ids))
	for _, id := range ids {
		if o.isIDInContractDataCache(id) {
			continue
		}
		idsToFetch = append(idsToFetch, id)
	}
	return idsToFetch
}

func (o *Manager) getCacheCollectionData(ids []thirdparty.ContractID) map[string]*thirdparty.CollectionData {
	o.collectionsDataCacheLock.RLock()
	defer o.collectionsDataCacheLock.RUnlock()

	collections := make(map[string]*thirdparty.CollectionData)
	for _, id := range ids {
		if collection, ok := o.collectionsDataCache[id.HashKey()]; ok {
			collections[id.HashKey()] = &collection
			continue
		}
	}
	return collections
}

func (o *Manager) setCacheCollectionData(data thirdparty.CollectionData) {
	o.collectionsDataCacheLock.Lock()
	defer o.collectionsDataCacheLock.Unlock()

	o.collectionsDataCache[data.ID.HashKey()] = data
}

func (o *Manager) getCacheFullCollectibleData(uniqueIDs []thirdparty.CollectibleUniqueID) []thirdparty.FullCollectibleData {
	ret := make([]thirdparty.FullCollectibleData, 0, len(uniqueIDs))

	collectiblesData := o.getCacheCollectiblesData(uniqueIDs)

	contractIDs := make([]thirdparty.ContractID, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		contractIDs = append(contractIDs, id.ContractID)
	}

	collectionsData := o.getCacheCollectionData(contractIDs)

	for _, id := range uniqueIDs {
		collectibleData := collectiblesData[id.HashKey()]
		if collectibleData == nil {
			// Use empty data, set only ID
			collectibleData = &thirdparty.CollectibleData{
				ID: id,
			}
		}

		fullData := thirdparty.FullCollectibleData{
			CollectibleData: *collectibleData,
			CollectionData:  collectionsData[id.ContractID.HashKey()],
		}
		ret = append(ret, fullData)
	}
	return ret
}
