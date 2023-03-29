package collectibles

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/collectibles"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
)

const requestTimeout = 5 * time.Second

type Manager struct {
	rpcClient        *rpc.Client
	metadataProvider thirdparty.NFTMetadataProvider
	openseaAPIKey    string
}

func NewManager(rpcClient *rpc.Client, metadataProvider thirdparty.NFTMetadataProvider, openseaAPIKey string) *Manager {
	return &Manager{
		rpcClient:        rpcClient,
		metadataProvider: metadataProvider,
		openseaAPIKey:    openseaAPIKey,
	}
}

func (o *Manager) FetchAllCollectionsByOwner(chainID uint64, owner common.Address) ([]opensea.OwnedCollection, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey)
	if err != nil {
		return nil, err
	}
	return client.FetchAllCollectionsByOwner(owner)
}

func (o *Manager) FetchAllAssetsByOwnerAndCollection(chainID uint64, owner common.Address, collectionSlug string, cursor string, limit int) (*opensea.AssetContainer, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey)
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

func (o *Manager) FetchAllAssetsByOwnerAndContractAddress(chainID uint64, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*opensea.AssetContainer, error) {
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey)
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
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey)
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
	client, err := opensea.NewOpenseaClient(chainID, o.openseaAPIKey)
	if err != nil {
		return nil, err
	}

	assetContainer, err := client.FetchAssetsByNFTUniqueID(uniqueIDs, limit)
	if err != nil {
		return nil, err
	}

	err = o.processAssets(chainID, assetContainer.Assets)
	if err != nil {
		return nil, err
	}

	return assetContainer, nil
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
		if strings.HasPrefix(err.Error(), "execution reverted") {
			// Contract doesn't support "TokenURI" method
			return "", nil
		}
		return "", err
	}

	return tokenURI, err
}

func (o *Manager) processAssets(chainID uint64, assets []opensea.Asset) error {
	for idx, asset := range assets {
		if isMetadataEmpty(asset) {
			id := thirdparty.NFTUniqueID{
				ContractAddress: common.HexToAddress(asset.Contract.Address),
				TokenID:         asset.TokenID,
			}

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
	}

	return nil
}
