package collectibles

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/contracts/collectibles"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
)

const requestTimeout = 5 * time.Second

const hystrixContractOwnershipClientName = "contractOwnershipClient"

const maxNFTDescriptionLength = 1024

// ERC721 does not support function "TokenURI" if call
// returns error starting with one of these strings
var noTokenURIErrorPrefixes = []string{
	"execution reverted",
	"abi: attempting to unmarshall",
}

type Manager struct {
	rpcClient                         *rpc.Client
	mainContractOwnershipProvider     thirdparty.NFTContractOwnershipProvider
	fallbackContractOwnershipProvider thirdparty.NFTContractOwnershipProvider
	metadataProvider                  thirdparty.NFTMetadataProvider
	openseaAPIKey                     string
	nftCache                          map[uint64]map[string]opensea.Asset
	nftCacheLock                      sync.RWMutex
	walletFeed                        *event.Feed
}

func NewManager(rpcClient *rpc.Client, mainContractOwnershipProvider thirdparty.NFTContractOwnershipProvider, fallbackContractOwnershipProvider thirdparty.NFTContractOwnershipProvider, openseaAPIKey string, walletFeed *event.Feed) *Manager {
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
		openseaAPIKey:                     openseaAPIKey,
		nftCache:                          make(map[uint64]map[string]opensea.Asset),
		walletFeed:                        walletFeed,
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
func (o *Manager) SetMetadataProvider(metadataProvider thirdparty.NFTMetadataProvider) {
	o.metadataProvider = metadataProvider
}

func (o *Manager) FetchAllCollectionsByOwner(chainID uint64, owner common.Address) ([]opensea.OwnedCollection, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey, o.walletFeed)
	if err != nil {
		return nil, err
	}
	return client.FetchAllCollectionsByOwner(owner)
}

func (o *Manager) FetchAllAssetsByOwnerAndCollection(chainID uint64, owner common.Address, collectionSlug string, cursor string, limit int) (*opensea.AssetContainer, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey, o.walletFeed)
	if err != nil {
		return nil, err
	}

	assetContainer, err := client.FetchAllAssetsByOwnerAndCollection(owner, collectionSlug, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(chainID, assetContainer.Assets)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

// Need to combine different providers to support all needed ChainIDs
func (o *Manager) FetchBalancesByOwnerAndContractAddress(chainID uint64, ownerAddress common.Address, contractAddresses []common.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	ret := make(thirdparty.TokenBalancesPerContractAddress)

	for _, contractAddress := range contractAddresses {
		ret[contractAddress] = make([]thirdparty.TokenBalance, 0)
	}

	// Try with more direct endpoint first (OpenSea)
	assetsContainer, err := o.FetchAllAssetsByOwnerAndContractAddress(chainID, ownerAddress, contractAddresses, "", 0)
	if err == opensea.ErrChainIDNotSupported {
		// Use contract ownership providers
		for _, contractAddress := range contractAddresses {
			ownership, err := o.FetchNFTOwnersByContractAddress(chainID, contractAddress)
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
		for _, asset := range assetsContainer.Assets {
			contractAddress := common.HexToAddress(asset.Contract.Address)
			balance := thirdparty.TokenBalance{
				TokenID: asset.TokenID,
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

func (o *Manager) FetchAllAssetsByOwnerAndContractAddress(chainID uint64, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*opensea.AssetContainer, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey, o.walletFeed)
	if err != nil {
		return nil, err
	}

	assetContainer, err := client.FetchAllAssetsByOwnerAndContractAddress(owner, contractAddresses, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(chainID, assetContainer.Assets)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchAllAssetsByOwner(chainID uint64, owner common.Address, cursor string, limit int) (*opensea.AssetContainer, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey, o.walletFeed)
	if err != nil {
		return nil, err
	}

	assetContainer, err := client.FetchAllAssetsByOwner(owner, cursor, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(chainID, assetContainer.Assets)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
}

func (o *Manager) FetchAssetsByNFTUniqueID(chainID uint64, uniqueIDs []thirdparty.NFTUniqueID, limit int) (*opensea.AssetContainer, error) {
	assetContainer := new(opensea.AssetContainer)

	idsToFetch := o.getIDsNotInCache(chainID, uniqueIDs)
	if len(idsToFetch) > 0 {
		client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey, o.walletFeed)
		if err != nil {
			return nil, err
		}

		fetchedAssetContainer, err := client.FetchAssetsByNFTUniqueID(idsToFetch, limit)
		if err != nil {
			return nil, err
		}

		err = o.processAssets(chainID, fetchedAssetContainer.Assets)
		if err != nil {
			return nil, err
		}

		assetContainer.NextCursor = fetchedAssetContainer.NextCursor
		assetContainer.PreviousCursor = fetchedAssetContainer.PreviousCursor
	}

	assetContainer.Assets = o.getCachedAssets(chainID, uniqueIDs)

	return assetContainer, nil
}

func (o *Manager) FetchNFTOwnersByContractAddress(chainID uint64, contractAddress common.Address) (*thirdparty.NFTContractOwnership, error) {
	mainFunc := func() (any, error) {
		return o.mainContractOwnershipProvider.FetchNFTOwnersByContractAddress(chainID, contractAddress)
	}
	var fallbackFunc func() (any, error) = nil
	if o.fallbackContractOwnershipProvider != nil && o.fallbackContractOwnershipProvider.IsChainSupported(chainID) {
		fallbackFunc = func() (any, error) {
			return o.fallbackContractOwnershipProvider.FetchNFTOwnersByContractAddress(chainID, contractAddress)
		}
	}
	owners, err := makeContractOwnershipCall(mainFunc, fallbackFunc)
	if err != nil {
		return nil, err
	}

	return owners.(*thirdparty.NFTContractOwnership), nil
}

func isMetadataEmpty(asset opensea.Asset) bool {
	return asset.Name == "" &&
		asset.Description == "" &&
		asset.ImageURL == "" &&
		asset.TokenURI == ""
}

func (o *Manager) fetchTokenURI(chainID uint64, id thirdparty.NFTUniqueID) (string, error) {
	backend, err := o.rpcClient.EthClient(chainID)
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

func (o *Manager) processAssets(chainID uint64, assets []opensea.Asset) error {
	o.nftCacheLock.Lock()
	defer o.nftCacheLock.Unlock()

	if _, ok := o.nftCache[chainID]; !ok {
		o.nftCache[chainID] = make(map[string]opensea.Asset)
	}

	for idx, asset := range assets {
		id := thirdparty.NFTUniqueID{
			ContractAddress: common.HexToAddress(asset.Contract.Address),
			TokenID:         asset.TokenID,
		}

		if isMetadataEmpty(asset) {
			tokenURI, err := o.fetchTokenURI(chainID, id)

			if err != nil {
				return err
			}

			assets[idx].TokenURI = tokenURI

			canProvide, err := o.metadataProvider.CanProvideNFTMetadata(chainID, id, tokenURI)

			if err != nil {
				return err
			}

			if canProvide {
				metadata, err := o.metadataProvider.FetchNFTMetadata(chainID, id, tokenURI)
				if err != nil {
					return err
				}

				if metadata != nil {
					assets[idx].Name = metadata.Name
					assets[idx].Description = metadata.Description
					assets[idx].Collection.ImageURL = metadata.CollectionImageURL
					assets[idx].ImageURL = metadata.ImageURL
				}
			}
		}

		// The NFT description field could be arbitrarily large, causing memory management issues upstream.
		// Trim it to a reasonable length here.
		if len(assets[idx].Description) > maxNFTDescriptionLength {
			assets[idx].Description = assets[idx].Description[:maxNFTDescriptionLength]
		}

		o.nftCache[chainID][id.HashKey()] = assets[idx]
	}

	return nil
}

func (o *Manager) getIDsNotInCache(chainID uint64, uniqueIDs []thirdparty.NFTUniqueID) []thirdparty.NFTUniqueID {
	o.nftCacheLock.RLock()
	defer o.nftCacheLock.RUnlock()

	idsToFetch := make([]thirdparty.NFTUniqueID, 0, len(uniqueIDs))
	if _, ok := o.nftCache[chainID]; !ok {
		idsToFetch = uniqueIDs
	} else {
		for _, id := range uniqueIDs {
			if _, ok := o.nftCache[chainID][id.HashKey()]; !ok {
				idsToFetch = append(idsToFetch, id)
			}
		}
	}
	return idsToFetch
}

func (o *Manager) getCachedAssets(chainID uint64, uniqueIDs []thirdparty.NFTUniqueID) []opensea.Asset {
	o.nftCacheLock.RLock()
	defer o.nftCacheLock.RUnlock()

	assets := make([]opensea.Asset, 0, len(uniqueIDs))

	if _, ok := o.nftCache[chainID]; ok {
		for _, id := range uniqueIDs {

			if asset, ok := o.nftCache[chainID][id.HashKey()]; ok {
				assets = append(assets, asset)
			}
		}
	}

	return assets
}
