package opensea

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	EventCollectibleStatusChanged walletevent.EventType = "wallet-collectible-status-changed"
)

const AssetLimit = 200
const CollectionLimit = 300

const RequestTimeout = 5 * time.Second
const GetRequestRetryMaxCount = 15
const GetRequestWaitTime = 300 * time.Millisecond

const ChainIDRequiringAPIKey = walletCommon.EthereumMainnet

const FetchNoLimit = 0

var (
	ErrChainIDNotSupported = errors.New("chainID not supported by opensea API")
)

type urlGetter func(walletCommon.ChainID, string) (string, error)

func getbaseURL(chainID walletCommon.ChainID) (string, error) {
	// v1 Endpoints only support L1 chain
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet:
		return "https://api.opensea.io/api/v1", nil
	case walletCommon.EthereumGoerli:
		return "https://testnets-api.opensea.io/api/v1", nil
	}

	return "", ErrChainIDNotSupported
}

func getURL(chainID walletCommon.ChainID, path string) (string, error) {
	baseURL, err := getbaseURL(chainID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", baseURL, path), nil
}

func chainStringToChainID(chainString string) walletCommon.ChainID {
	chainID := walletCommon.UnknownChainID
	switch chainString {
	case "ethereum":
		chainID = walletCommon.EthereumMainnet
	case "arbitrum":
		chainID = walletCommon.ArbitrumMainnet
	case "optimism":
		chainID = walletCommon.OptimismMainnet
	case "goerli":
		chainID = walletCommon.EthereumGoerli
	case "arbitrum_goerli":
		chainID = walletCommon.ArbitrumGoerli
	case "optimism_goerli":
		chainID = walletCommon.OptimismGoerli
	}
	return walletCommon.ChainID(chainID)
}

type TraitValue string

func (st *TraitValue) UnmarshalJSON(b []byte) error {
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}

	switch v := item.(type) {
	case float64:
		*st = TraitValue(strconv.FormatFloat(v, 'f', 2, 64))
	case int:
		*st = TraitValue(strconv.Itoa(v))
	case string:
		*st = TraitValue(v)

	}
	return nil
}

type AssetContainer struct {
	Assets         []Asset `json:"assets"`
	NextCursor     string  `json:"next"`
	PreviousCursor string  `json:"previous"`
}

type Contract struct {
	Address         string `json:"address"`
	ChainIdentifier string `json:"chain_identifier"`
}

type Trait struct {
	TraitType   string     `json:"trait_type"`
	Value       TraitValue `json:"value"`
	DisplayType string     `json:"display_type"`
	MaxValue    string     `json:"max_value"`
}

type PaymentToken struct {
	ID       int    `json:"id"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	ImageURL string `json:"image_url"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	EthPrice string `json:"eth_price"`
	UsdPrice string `json:"usd_price"`
}

type LastSale struct {
	PaymentToken PaymentToken `json:"payment_token"`
}

type SellOrder struct {
	CurrentPrice string `json:"current_price"`
}

type Asset struct {
	ID                 int            `json:"id"`
	TokenID            *bigint.BigInt `json:"token_id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Permalink          string         `json:"permalink"`
	ImageThumbnailURL  string         `json:"image_thumbnail_url"`
	ImageURL           string         `json:"image_url"`
	AnimationURL       string         `json:"animation_url"`
	AnimationMediaType string         `json:"animation_media_type"`
	Contract           Contract       `json:"asset_contract"`
	Collection         Collection     `json:"collection"`
	Traits             []Trait        `json:"traits"`
	LastSale           LastSale       `json:"last_sale"`
	SellOrders         []SellOrder    `json:"sell_orders"`
	BackgroundColor    string         `json:"background_color"`
	TokenURI           string         `json:"token_metadata"`
}

type CollectionTrait struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Collection struct {
	Name     string                     `json:"name"`
	Slug     string                     `json:"slug"`
	ImageURL string                     `json:"image_url"`
	Traits   map[string]CollectionTrait `json:"traits"`
}

type OwnedCollection struct {
	Collection
	OwnedAssetCount *bigint.BigInt `json:"owned_asset_count"`
}

func (c *Asset) id() thirdparty.CollectibleUniqueID {
	return thirdparty.CollectibleUniqueID{
		ChainID:         chainStringToChainID(c.Contract.ChainIdentifier),
		ContractAddress: common.HexToAddress(c.Contract.Address),
		TokenID:         c.TokenID,
	}
}

func openseaToCollectibleTraits(traits []Trait) []thirdparty.CollectibleTrait {
	ret := make([]thirdparty.CollectibleTrait, 0, len(traits))
	caser := cases.Title(language.Und, cases.NoLower)
	for _, orig := range traits {
		dest := thirdparty.CollectibleTrait{
			TraitType:   strings.Replace(orig.TraitType, "_", " ", 1),
			Value:       caser.String(string(orig.Value)),
			DisplayType: orig.DisplayType,
			MaxValue:    orig.MaxValue,
		}

		ret = append(ret, dest)
	}
	return ret
}

func (c *Collection) toCommon() thirdparty.CollectionData {
	ret := thirdparty.CollectionData{
		Name:     c.Name,
		Slug:     c.Slug,
		ImageURL: c.ImageURL,
		Traits:   make(map[string]thirdparty.CollectionTrait),
	}
	for traitType, trait := range c.Traits {
		ret.Traits[traitType] = thirdparty.CollectionTrait{
			Min: trait.Min,
			Max: trait.Max,
		}
	}
	return ret
}

func (c *Asset) toCommon() thirdparty.CollectibleData {
	return thirdparty.CollectibleData{
		ID:                 c.id(),
		Name:               c.Name,
		Description:        c.Description,
		Permalink:          c.Permalink,
		ImageURL:           c.ImageURL,
		AnimationURL:       c.AnimationURL,
		AnimationMediaType: c.AnimationMediaType,
		Traits:             openseaToCollectibleTraits(c.Traits),
		BackgroundColor:    c.BackgroundColor,
		TokenURI:           c.TokenURI,
		CollectionData:     c.Collection.toCommon(),
	}
}

type HTTPClient struct {
	client         *http.Client
	getRequestLock sync.RWMutex
}

func newHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

func (o *HTTPClient) doGetRequest(url string, apiKey string) ([]byte, error) {
	// Ensure only one thread makes a request at a time
	o.getRequestLock.Lock()
	defer o.getRequestLock.Unlock()

	retryCount := 0
	statusCode := http.StatusOK

	// Try to do the request without an apiKey first
	tmpAPIKey := ""

	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0")
		if len(tmpAPIKey) > 0 {
			req.Header.Set("X-API-KEY", tmpAPIKey)
		}

		resp, err := o.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Error("failed to close opensea request body", "err", err)
			}
		}()

		statusCode = resp.StatusCode
		switch resp.StatusCode {
		case http.StatusOK:
			body, err := ioutil.ReadAll(resp.Body)
			return body, err
		case http.StatusTooManyRequests:
			if retryCount < GetRequestRetryMaxCount {
				// sleep and retry
				time.Sleep(GetRequestWaitTime)
				retryCount++
				continue
			}
			// break and error
		case http.StatusForbidden:
			// Request requires an apiKey, set it and retry
			if tmpAPIKey == "" && apiKey != "" {
				tmpAPIKey = apiKey
				// sleep and retry
				time.Sleep(GetRequestWaitTime)
				continue
			}
			// break and error
		default:
			// break and error
		}
		break
	}
	return nil, fmt.Errorf("unsuccessful request: %d %s", statusCode, http.StatusText(statusCode))
}

func (o *HTTPClient) doContentTypeRequest(url string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := o.client.Do(req)
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

type Client struct {
	client           *HTTPClient
	apiKey           string
	connectionStatus *connection.Status
	urlGetter        urlGetter
}

// new opensea client.
func NewClient(apiKey string, feed *event.Feed) *Client {
	return &Client{
		client:           newHTTPClient(),
		apiKey:           apiKey,
		connectionStatus: connection.NewStatus(EventCollectibleStatusChanged, feed),
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

func (o *Client) FetchAllAssetsByOwnerAndCollection(chainID walletCommon.ChainID, owner common.Address, collectionSlug string, cursor string, limit int) (*thirdparty.CollectibleDataContainer, error) {
	queryParams := url.Values{
		"owner":      {owner.String()},
		"collection": {collectionSlug},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(chainID, queryParams, limit)
}

func (o *Client) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.CollectibleDataContainer, error) {
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

func (o *Client) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.CollectibleDataContainer, error) {
	queryParams := url.Values{
		"owner": {owner.String()},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(chainID, queryParams, limit)
}

func (o *Client) FetchAssetsByCollectibleUniqueID(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.CollectibleData, error) {
	queryParams := url.Values{}

	ret := make([]thirdparty.CollectibleData, 0, len(uniqueIDs))

	idsPerChainID := thirdparty.GroupCollectibleUIDsByChainID(uniqueIDs)
	for chainID, ids := range idsPerChainID {
		for _, id := range ids {
			queryParams.Add("token_ids", id.TokenID.String())
			queryParams.Add("asset_contract_addresses", id.ContractAddress.String())
		}

		data, err := o.fetchAssets(chainID, queryParams, FetchNoLimit)
		if err != nil {
			return nil, err
		}

		ret = append(ret, data.Collectibles...)
	}

	return ret, nil
}

func (o *Client) fetchAssets(chainID walletCommon.ChainID, queryParams url.Values, limit int) (*thirdparty.CollectibleDataContainer, error) {
	assets := new(thirdparty.CollectibleDataContainer)

	if len(queryParams["cursor"]) > 0 {
		assets.PreviousCursor = queryParams["cursor"][0]
	}

	tmpLimit := AssetLimit
	if limit > FetchNoLimit && limit < tmpLimit {
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
			if len(asset.AnimationURL) > 0 {
				asset.AnimationMediaType, err = o.client.doContentTypeRequest(asset.AnimationURL)
				if err != nil {
					asset.AnimationURL = ""
				}
			}
			assets.Collectibles = append(assets.Collectibles, asset.toCommon())
		}
		assets.NextCursor = container.NextCursor

		if len(assets.NextCursor) == 0 {
			break
		}

		queryParams["cursor"] = []string{assets.NextCursor}

		if limit > FetchNoLimit && len(assets.Collectibles) >= limit {
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

	baseURL, err := getbaseURL(chainID)

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
			if len(asset.AnimationURL) > 0 {
				asset.AnimationMediaType, err = o.client.doContentTypeRequest(asset.AnimationURL)
				if err != nil {
					asset.AnimationURL = ""
				}
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
