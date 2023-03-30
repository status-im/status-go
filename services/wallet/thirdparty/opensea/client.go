package opensea

import (
	"encoding/json"
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

	"github.com/status-im/status-go/services/wallet/bigint"
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

const ChainIDRequiringAPIKey = 1

func getbaseURL(chainID uint64) (string, error) {
	switch chainID {
	case 1:
		return "https://api.opensea.io/api/v1", nil
	case 4:
		return "https://rinkeby-api.opensea.io/api/v1", nil
	case 5:
		return "https://testnets-api.opensea.io/api/v1", nil
	}

	return "", fmt.Errorf("chainID not supported")
}

var OpenseaClientInstances = make(map[uint64]*Client)
var OpenseaHTTPClient *HTTPClient = nil

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
	Address string `json:"address"`
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
	ID                int            `json:"id"`
	TokenID           *bigint.BigInt `json:"token_id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Permalink         string         `json:"permalink"`
	ImageThumbnailURL string         `json:"image_thumbnail_url"`
	ImageURL          string         `json:"image_url"`
	Contract          Contract       `json:"asset_contract"`
	Collection        Collection     `json:"collection"`
	Traits            []Trait        `json:"traits"`
	LastSale          LastSale       `json:"last_sale"`
	SellOrders        []SellOrder    `json:"sell_orders"`
	BackgroundColor   string         `json:"background_color"`
	TokenURI          string         `json:"token_metadata"`
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

type Client struct {
	client          *HTTPClient
	url             string
	apiKey          string
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
	feed            *event.Feed
}

// new opensea client.
func NewOpenseaClient(chainID uint64, apiKey string, feed *event.Feed) (*Client, error) {
	if OpenseaHTTPClient == nil {
		OpenseaHTTPClient = newHTTPClient()
	}

	var tmpAPIKey string = ""
	if chainID == ChainIDRequiringAPIKey {
		tmpAPIKey = apiKey
	}
	if client, ok := OpenseaClientInstances[chainID]; ok {
		if client.apiKey == tmpAPIKey {
			return client, nil
		}
	}

	baseURL, err := getbaseURL(chainID)
	if err != nil {
		return nil, err
	}

	openseaClient := &Client{
		client:        OpenseaHTTPClient,
		url:           baseURL,
		apiKey:        tmpAPIKey,
		IsConnected:   true,
		LastCheckedAt: time.Now().Unix(),
		feed:          feed,
	}
	OpenseaClientInstances[chainID] = openseaClient
	return openseaClient, nil
}

func (o *Client) setIsConnected(value bool) {
	o.IsConnectedLock.Lock()
	defer o.IsConnectedLock.Unlock()
	o.LastCheckedAt = time.Now().Unix()
	if value != o.IsConnected {
		message := "down"
		if value {
			message = "up"
		}
		if o.feed != nil {
			o.feed.Send(walletevent.Event{
				Type:     EventCollectibleStatusChanged,
				Accounts: []common.Address{},
				Message:  message,
				At:       time.Now().Unix(),
			})
		}
	}
	o.IsConnected = value
}
func (o *Client) FetchAllCollectionsByOwner(owner common.Address) ([]OwnedCollection, error) {
	offset := 0
	var collections []OwnedCollection
	for {
		url := fmt.Sprintf("%s/collections?asset_owner=%s&offset=%d&limit=%d", o.url, owner, offset, CollectionLimit)
		body, err := o.client.doGetRequest(url, o.apiKey)
		if err != nil {
			o.setIsConnected(false)
			return nil, err
		}

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		var tmp []OwnedCollection
		err = json.Unmarshal(body, &tmp)
		if err != nil {
			o.setIsConnected(false)
			return nil, err
		}

		collections = append(collections, tmp...)

		if len(tmp) < CollectionLimit {
			break
		}
	}
	o.setIsConnected(true)
	return collections, nil
}

func (o *Client) FetchAllAssetsByOwnerAndCollection(owner common.Address, collectionSlug string, cursor string, limit int) (*AssetContainer, error) {
	queryParams := url.Values{
		"owner":      {owner.String()},
		"collection": {collectionSlug},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(queryParams, limit)
}

func (o *Client) FetchAllAssetsByOwnerAndContractAddress(owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*AssetContainer, error) {
	queryParams := url.Values{
		"owner": {owner.String()},
	}

	for _, contractAddress := range contractAddresses {
		queryParams.Add("asset_contract_addresses", contractAddress.String())
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(queryParams, limit)
}

func (o *Client) FetchAllAssetsByOwner(owner common.Address, cursor string, limit int) (*AssetContainer, error) {
	queryParams := url.Values{
		"owner": {owner.String()},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(queryParams, limit)
}

func (o *Client) FetchAssetsByNFTUniqueID(uniqueIDs []thirdparty.NFTUniqueID, limit int) (*AssetContainer, error) {
	queryParams := url.Values{}

	for _, uniqueID := range uniqueIDs {
		queryParams.Add("token_ids", uniqueID.TokenID.String())
		queryParams.Add("asset_contract_addresses", uniqueID.ContractAddress.String())
	}

	return o.fetchAssets(queryParams, limit)
}

func (o *Client) fetchAssets(queryParams url.Values, limit int) (*AssetContainer, error) {
	assets := new(AssetContainer)

	if len(queryParams["cursor"]) > 0 {
		assets.PreviousCursor = queryParams["cursor"][0]
	}

	tmpLimit := AssetLimit
	if limit > 0 && limit < tmpLimit {
		tmpLimit = AssetLimit
	}

	queryParams["limit"] = []string{strconv.Itoa(tmpLimit)}
	for {
		url := o.url + "/assets?" + queryParams.Encode()

		body, err := o.client.doGetRequest(url, o.apiKey)
		if err != nil {
			o.setIsConnected(false)
			return nil, err
		}

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		container := AssetContainer{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			o.setIsConnected(false)
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

	o.setIsConnected(true)
	return assets, nil
}
