package collectibles

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/collectibles"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
)

const FetchNoLimit = 0
const FetchFromStartCursor = ""

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
	nftCache                          map[walletCommon.ChainID]map[string]thirdparty.CollectibleData
	nftCacheLock                      sync.RWMutex
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
		nftCache:                          make(map[walletCommon.ChainID]map[string]thirdparty.CollectibleData),
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

func (o *Manager) FetchAllAssetsByOwnerAndCollection(chainID walletCommon.ChainID, owner common.Address, collectionSlug string, cursor string, limit int) (*thirdparty.CollectibleDataContainer, error) {
	assetContainer, err := o.opensea.FetchAllAssetsByOwnerAndCollection(chainID, owner, collectionSlug, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(assetContainer.Collectibles)
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
	assetsContainer, err := o.FetchAllAssetsByOwnerAndContractAddress(chainID, ownerAddress, contractAddresses, FetchFromStartCursor, FetchNoLimit)
	if err == opensea.ErrChainIDNotSupported {
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
		for _, collectible := range assetsContainer.Collectibles {
			contractAddress := collectible.ID.ContractAddress
			balance := thirdparty.TokenBalance{
				TokenID: collectible.ID.TokenID,
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

func (o *Manager) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.CollectibleDataContainer, error) {
	assetContainer, err := o.opensea.FetchAllAssetsByOwnerAndContractAddress(chainID, owner, contractAddresses, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(assetContainer.Collectibles)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.CollectibleDataContainer, error) {
	assetContainer, err := o.opensea.FetchAllAssetsByOwner(chainID, owner, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(assetContainer.Collectibles)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchCollectibleOwnershipByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.CollectibleOwnershipContainer, error) {
	assetContainer, err := o.FetchAllAssetsByOwner(chainID, owner, cursor, limit)
	if err != nil {
		return nil, err
	}

	ret := assetContainer.ToOwnershipContainer()

	return &ret, nil
}

func (o *Manager) FetchAssetsByCollectibleUniqueID(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.CollectibleData, error) {
	idsToFetch := o.getIDsNotInCollectiblesDataCache(uniqueIDs)
	if len(idsToFetch) > 0 {
		fetchedAssets, err := o.opensea.FetchAssetsByCollectibleUniqueID(idsToFetch)
		if err != nil {
			return nil, err
		}

		err = o.processAssets(fetchedAssets)
		if err != nil {
			return nil, err
		}

	}

	return o.getCacheCollectiblesData(uniqueIDs), nil
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
	backend, err := o.rpcClient.EthClient(uint64(id.ChainID))
	if err != nil {
		return "", err
	}

	caller, err := collectibles.NewCollectiblesCaller(id.ContractAddress, backend)
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

func (o *Manager) processAssets(assets []thirdparty.CollectibleData) error {
	for idx, asset := range assets {
		id := asset.ID

		if isMetadataEmpty(asset) {
			if o.metadataProvider == nil {
				return fmt.Errorf("CollectibleMetadataProvider not available")
			}
			tokenURI, err := o.fetchTokenURI(id)

			if err != nil {
				return err
			}

			assets[idx].TokenURI = tokenURI

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

		o.setCacheCollectibleData(assets[idx])
	}

	return nil
}

func (o *Manager) isIDInCollectiblesDataCache(id thirdparty.CollectibleUniqueID) bool {
	o.nftCacheLock.RLock()
	defer o.nftCacheLock.RUnlock()
	if _, ok := o.nftCache[id.ChainID]; ok {
		if _, ok := o.nftCache[id.ChainID][id.HashKey()]; ok {
			return true
		}
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

func (o *Manager) getCacheCollectiblesData(uniqueIDs []thirdparty.CollectibleUniqueID) []thirdparty.CollectibleData {
	o.nftCacheLock.RLock()
	defer o.nftCacheLock.RUnlock()

	assets := make([]thirdparty.CollectibleData, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		if _, ok := o.nftCache[id.ChainID]; ok {
			if asset, ok := o.nftCache[id.ChainID][id.HashKey()]; ok {
				assets = append(assets, asset)
				continue
			}
		}
		emptyAsset := thirdparty.CollectibleData{
			ID: id,
		}
		assets = append(assets, emptyAsset)
	}
	return assets
}

func (o *Manager) setCacheCollectibleData(data thirdparty.CollectibleData) {
	o.nftCacheLock.Lock()
	defer o.nftCacheLock.Unlock()

	id := data.ID

	if _, ok := o.nftCache[id.ChainID]; !ok {
		o.nftCache[id.ChainID] = make(map[string]thirdparty.CollectibleData)
	}

	o.nftCache[id.ChainID][id.HashKey()] = data
}
