package opensea

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const AssetLimit = 200
const CollectionLimit = 300

const ChainIDRequiringAPIKey = walletCommon.EthereumMainnet

func getBaseURL(chainID walletCommon.ChainID) (string, error) {
	// v1 Endpoints only support L1 chain
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet:
		return "https://api.opensea.io/api/v1", nil
	case walletCommon.EthereumSepolia:
		return "https://testnets-api.opensea.io/api/v1", nil
	}

	return "", thirdparty.ErrChainIDNotSupported
}

func (o *Client) ID() string {
	return OpenseaV1ID
}

func (o *Client) IsChainSupported(chainID walletCommon.ChainID) bool {
	_, err := getBaseURL(chainID)
	return err == nil
}

func (o *Client) IsConnected() bool {
	return o.connectionStatus.IsConnected()
}

func getURL(chainID walletCommon.ChainID, path string) (string, error) {
	baseURL, err := getBaseURL(chainID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", baseURL, path), nil
}

type Client struct {
	client           *HTTPClient
	apiKey           string
	connectionStatus *connection.Status
	urlGetter        urlGetter
}

// new opensea v1 client.
func NewClient(apiKey string, httpClient *HTTPClient) *Client {
	if apiKey == "" {
		log.Warn("OpenseaV1 API key not available")
	}

	return &Client{
		client:           httpClient,
		apiKey:           apiKey,
		connectionStatus: connection.NewStatus(),
		urlGetter:        getURL,
	}
}

func (o *Client) FetchAllCollectionsByOwner(chainID walletCommon.ChainID, owner common.Address) ([]OwnedCollection, error) {
	offset := 0
	var collections []OwnedCollection

	for {
		path := fmt.Sprintf("collections?asset_owner=%s&offset=%d&limit=%d", owner, offset, CollectionLimit)
		url, err := o.urlGetter(chainID, path)
		if err != nil {
			return nil, err
		}

		body, err := o.client.doGetRequest(url, o.apiKey)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		var tmp []OwnedCollection
		err = json.Unmarshal(body, &tmp)
		if err != nil {
			return nil, err
		}

		collections = append(collections, tmp...)

		if len(tmp) < CollectionLimit {
			break
		}
	}
	return collections, nil
}

func (o *Client) FetchCollectionsDataByContractID(ids []thirdparty.ContractID) ([]thirdparty.CollectionData, error) {
	ret := make([]thirdparty.CollectionData, 0, len(ids))

	for _, id := range ids {
		path := fmt.Sprintf("asset_contract/%s", id.Address.String())
		url, err := o.urlGetter(id.ChainID, path)
		if err != nil {
			return nil, err
		}

		body, err := o.client.doGetRequest(url, o.apiKey)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		var tmp AssetContract
		err = json.Unmarshal(body, &tmp)
		if err != nil {
			return nil, err
		}

		ret = append(ret, tmp.Collection.toCollectionData(id))
	}

	return ret, nil
}

func (o *Client) FetchAllAssetsByOwnerAndCollection(chainID walletCommon.ChainID, owner common.Address, collectionSlug string, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{
		"owner":      {owner.String()},
		"collection": {collectionSlug},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(chainID, queryParams, limit)
}

func (o *Client) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{
		"owner": {owner.String()},
	}

	for _, contractAddress := range contractAddresses {
		queryParams.Add("asset_contract_addresses", contractAddress.String())
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(chainID, queryParams, limit)
}

func (o *Client) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{
		"owner": {owner.String()},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(chainID, queryParams, limit)
}

func (o *Client) FetchAssetsByCollectibleUniqueID(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	queryParams := url.Values{}

	ret := make([]thirdparty.FullCollectibleData, 0, len(uniqueIDs))

	idsPerChainID := thirdparty.GroupCollectibleUIDsByChainID(uniqueIDs)
	for chainID, ids := range idsPerChainID {
		for _, id := range ids {
			queryParams.Add("token_ids", id.TokenID.String())
			queryParams.Add("asset_contract_addresses", id.ContractID.Address.String())
		}

		data, err := o.fetchAssets(chainID, queryParams, thirdparty.FetchNoLimit)
		if err != nil {
			return nil, err
		}

		ret = append(ret, data.Items...)
	}

	return ret, nil
}

func (o *Client) fetchAssets(chainID walletCommon.ChainID, queryParams url.Values, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assets := new(thirdparty.FullCollectibleDataContainer)

	if len(queryParams["cursor"]) > 0 {
		assets.PreviousCursor = queryParams["cursor"][0]
	}

	tmpLimit := AssetLimit
	if limit > thirdparty.FetchNoLimit && limit < tmpLimit {
		tmpLimit = limit
	}

	queryParams["limit"] = []string{strconv.Itoa(tmpLimit)}
	for {
		path := "assets?" + queryParams.Encode()
		url, err := o.urlGetter(chainID, path)
		if err != nil {
			return nil, err
		}

		body, err := o.client.doGetRequest(url, o.apiKey)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		container := AssetContainer{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			return nil, err
		}

		for _, asset := range container.Assets {
			assets.Items = append(assets.Items, asset.toCommon())
		}
		assets.NextCursor = container.NextCursor

		if len(assets.NextCursor) == 0 {
			break
		}

		queryParams["cursor"] = []string{assets.NextCursor}

		if limit > thirdparty.FetchNoLimit && len(assets.Items) >= limit {
			break
		}
	}

	return assets, nil
}

// Only here for compatibility with mobile app, to be removed
func (o *Client) FetchAllOpenseaAssetsByOwnerAndCollection(chainID walletCommon.ChainID, owner common.Address, collectionSlug string, cursor string, limit int) (*AssetContainer, error) {
	queryParams := url.Values{
		"owner":      {owner.String()},
		"collection": {collectionSlug},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchOpenseaAssets(chainID, queryParams, limit)
}

func (o *Client) fetchOpenseaAssets(chainID walletCommon.ChainID, queryParams url.Values, limit int) (*AssetContainer, error) {
	assets := new(AssetContainer)

	if len(queryParams["cursor"]) > 0 {
		assets.PreviousCursor = queryParams["cursor"][0]
	}

	tmpLimit := AssetLimit
	if limit > 0 && limit < tmpLimit {
		tmpLimit = limit
	}

	baseURL, err := getBaseURL(chainID)

	if err != nil {
		return nil, err
	}

	queryParams["limit"] = []string{strconv.Itoa(tmpLimit)}
	for {
		url := baseURL + "/assets?" + queryParams.Encode()

		body, err := o.client.doGetRequest(url, o.apiKey)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		container := AssetContainer{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			return nil, err
		}

		for _, asset := range container.Assets {
			for i := range asset.Traits {
				asset.Traits[i].TraitType = strings.Replace(asset.Traits[i].TraitType, "_", " ", 1)
				asset.Traits[i].Value = TraitValue(strings.Title(string(asset.Traits[i].Value)))
			}
			assets.Assets = append(assets.Assets, asset)
		}
		assets.NextCursor = container.NextCursor

		if len(assets.NextCursor) == 0 {
			break
		}

		queryParams["cursor"] = []string{assets.NextCursor}

		if limit > 0 && len(assets.Assets) >= limit {
			break
		}
	}

	return assets, nil
}
