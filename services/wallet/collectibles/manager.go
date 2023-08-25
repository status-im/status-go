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
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
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

var (
	ErrNoProvidersAvailableForChainID = errors.New("no providers available for chainID")
)

type Manager struct {
	rpcClient                  *rpc.Client
	contractOwnershipProviders []thirdparty.CollectibleContractOwnershipProvider
	accountOwnershipProviders  []thirdparty.CollectibleAccountOwnershipProvider
	collectibleDataProviders   []thirdparty.CollectibleDataProvider
	collectionDataProviders    []thirdparty.CollectionDataProvider
	metadataProvider           thirdparty.CollectibleMetadataProvider
	opensea                    *opensea.Client
	httpClient                 *http.Client
	collectiblesDataDB         *CollectibleDataDB
	collectionsDataDB          *CollectionDataDB
}

func NewManager(db *sql.DB, rpcClient *rpc.Client, contractOwnershipProviders []thirdparty.CollectibleContractOwnershipProvider, accountOwnershipProviders []thirdparty.CollectibleAccountOwnershipProvider, collectibleDataProviders []thirdparty.CollectibleDataProvider, collectionDataProviders []thirdparty.CollectionDataProvider, opensea *opensea.Client) *Manager {
	hystrix.ConfigureCommand(hystrixContractOwnershipClientName, hystrix.CommandConfig{
		Timeout:               10000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	return &Manager{
		rpcClient:                  rpcClient,
		contractOwnershipProviders: contractOwnershipProviders,
		accountOwnershipProviders:  accountOwnershipProviders,
		collectibleDataProviders:   collectibleDataProviders,
		collectionDataProviders:    collectionDataProviders,
		opensea:                    opensea,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		collectiblesDataDB: NewCollectibleDataDB(db),
		collectionsDataDB:  NewCollectionDataDB(db),
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

	// Try with account ownership providers first
	assetsContainer, err := o.FetchAllAssetsByOwnerAndContractAddress(chainID, ownerAddress, contractAddresses, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	if err == ErrNoProvidersAvailableForChainID {
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

func (o *Manager) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	for _, provider := range o.accountOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}

		assetContainer, err := provider.FetchAllAssetsByOwnerAndContractAddress(chainID, owner, contractAddresses, cursor, limit)
		if err != nil {
			return nil, err
		}

		err = o.processFullCollectibleData(assetContainer.Items)
		if err != nil {
			return nil, err
		}

		return assetContainer, nil
	}

	return nil, ErrNoProvidersAvailableForChainID
}

func (o *Manager) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	for _, provider := range o.accountOwnershipProviders {
		if !provider.IsChainSupported(chainID) {
			continue
		}

		assetContainer, err := provider.FetchAllAssetsByOwner(chainID, owner, cursor, limit)
		if err != nil {
			return nil, err
		}

		err = o.processFullCollectibleData(assetContainer.Items)
		if err != nil {
			return nil, err
		}

		return assetContainer, nil
	}

	return nil, ErrNoProvidersAvailableForChainID
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
	missingIDs, err := o.collectiblesDataDB.GetIDsNotInDB(uniqueIDs)
	if err != nil {
		return nil, err
	}

	missingIDsPerChainID := thirdparty.GroupCollectibleUIDsByChainID(missingIDs)

	for chainID, idsToFetch := range missingIDsPerChainID {
		for _, provider := range o.collectibleDataProviders {
			if !provider.IsChainSupported(chainID) {
				continue
			}

			fetchedAssets, err := provider.FetchAssetsByCollectibleUniqueID(idsToFetch)
			if err != nil {
				return nil, err
			}

			err = o.processFullCollectibleData(fetchedAssets)
			if err != nil {
				return nil, err
			}

			break
		}
	}

	return o.getCacheFullCollectibleData(uniqueIDs)
}

func (o *Manager) FetchCollectionsDataByContractID(ids []thirdparty.ContractID) ([]thirdparty.CollectionData, error) {
	missingIDs, err := o.collectionsDataDB.GetIDsNotInDB(ids)
	if err != nil {
		return nil, err
	}

	missingIDsPerChainID := thirdparty.GroupContractIDsByChainID(missingIDs)

	for chainID, idsToFetch := range missingIDsPerChainID {
		for _, provider := range o.collectionDataProviders {
			if !provider.IsChainSupported(chainID) {
				continue
			}

			fetchedCollections, err := provider.FetchCollectionsDataByContractID(idsToFetch)
			if err != nil {
				return nil, err
			}

			err = o.processCollectionData(fetchedCollections)
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

func getCollectibleOwnersByContractAddressFunc(chainID walletCommon.ChainID, contractAddress common.Address, provider thirdparty.CollectibleContractOwnershipProvider) func() (any, error) {
	if provider == nil {
		return nil
	}
	return func() (any, error) {
		return provider.FetchCollectibleOwnersByContractAddress(chainID, contractAddress)
	}
}

func (o *Manager) FetchCollectibleOwnersByContractAddress(chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	mainProvider, fallbackProvider := o.getContractOwnershipProviders(chainID)
	if mainProvider == nil {
		return nil, ErrNoProvidersAvailableForChainID
	}

	mainFn := getCollectibleOwnersByContractAddressFunc(chainID, contractAddress, mainProvider)
	fallbackFn := getCollectibleOwnersByContractAddressFunc(chainID, contractAddress, fallbackProvider)

	owners, err := makeContractOwnershipCall(mainFn, fallbackFn)
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
	collectiblesData := make([]thirdparty.CollectibleData, 0, len(assets))
	collectionsData := make([]thirdparty.CollectionData, 0, len(assets))
	missingCollectionIDs := make([]thirdparty.ContractID, 0)

	for _, asset := range assets {
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

			asset.CollectibleData.TokenURI = tokenURI

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
					asset = *metadata
				}
			}
		}

		// Get Animation MediaType
		if len(asset.CollectibleData.AnimationURL) > 0 {
			contentType, err := o.doContentTypeRequest(asset.CollectibleData.AnimationURL)
			if err != nil {
				asset.CollectibleData.AnimationURL = ""
			}
			asset.CollectibleData.AnimationMediaType = contentType
		}

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

	err = o.collectionsDataDB.SetData(collectionsData)
	if err != nil {
		return err
	}

	if len(missingCollectionIDs) > 0 {
		// Calling this ensures collection data is fetched and cached (if not already available)
		_, err := o.FetchCollectionsDataByContractID(missingCollectionIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Manager) processCollectionData(collections []thirdparty.CollectionData) error {
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

		fullData := thirdparty.FullCollectibleData{
			CollectibleData: collectibleData,
			CollectionData:  &collectionData,
		}
		ret = append(ret, fullData)
	}

	return ret, nil
}
