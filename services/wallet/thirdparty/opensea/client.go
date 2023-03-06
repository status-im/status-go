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
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/services/wallet/bigint"
)

const AssetLimit = 200
const CollectionLimit = 300

const RequestRetryMaxCount = 1
const RequestWaitTime = 300 * time.Millisecond

var OpenseaClientInstances = make(map[uint64]*Client)

var BaseURLs = map[uint64]string{
	1: "https://api.opensea.io/api/v1",
	4: "https://rinkeby-api.opensea.io/api/v1",
	5: "https://testnets-api.opensea.io/api/v1",
}

const ChainIDRequiringAPIKey = 1

type TraitValue string

type NFTUniqueID struct {
	ContractAddress common.Address `json:"contractAddress"`
	TokenID         bigint.BigInt  `json:"tokenID"`
}

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

type Client struct {
	client          *http.Client
	url             string
	apiKey          string
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
	requestLock     sync.RWMutex
}

// new opensea client.
func NewOpenseaClient(chainID uint64, apiKey string) (*Client, error) {
	var tmpAPIKey string = ""
	if chainID == ChainIDRequiringAPIKey {
		tmpAPIKey = apiKey
	}
	if client, ok := OpenseaClientInstances[chainID]; ok {
		if client.apiKey == tmpAPIKey {
			return client, nil
		}
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}
	if url, ok := BaseURLs[chainID]; ok {
		openseaClient := &Client{
			client:        client,
			url:           url,
			apiKey:        tmpAPIKey,
			IsConnected:   true,
			LastCheckedAt: time.Now().Unix(),
		}
		OpenseaClientInstances[chainID] = openseaClient
		return openseaClient, nil
	}

	return nil, errors.New("ChainID not supported")
}

func (o *Client) setConnected(value bool) {
	o.IsConnectedLock.Lock()
	defer o.IsConnectedLock.Unlock()
	o.IsConnected = value
	o.LastCheckedAt = time.Now().Unix()
}

func (o *Client) FetchAllCollectionsByOwner(owner common.Address) ([]OwnedCollection, error) {
	offset := 0
	var collections []OwnedCollection
	for {
		url := fmt.Sprintf("%s/collections?asset_owner=%s&offset=%d&limit=%d", o.url, owner, offset, CollectionLimit)
		body, err := o.doOpenseaRequest(url)
		if err != nil {
			o.setConnected(false)
			return nil, err
		}

		var tmp []OwnedCollection
		err = json.Unmarshal(body, &tmp)
		if err != nil {
			o.setConnected(false)
			return nil, err
		}

		collections = append(collections, tmp...)

		if len(tmp) < CollectionLimit {
			break
		}
	}
	o.setConnected(true)
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

func (o *Client) FetchAllAssetsByOwner(owner common.Address, cursor string, limit int) (*AssetContainer, error) {
	queryParams := url.Values{
		"owner": {owner.String()},
	}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchAssets(queryParams, limit)
}

func (o *Client) FetchAssetsByNFTUniqueID(uniqueIDs []NFTUniqueID, limit int) (*AssetContainer, error) {
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

	tmpLimit := limit
	if AssetLimit < limit {
		tmpLimit = AssetLimit
	}

	queryParams["limit"] = []string{strconv.Itoa(tmpLimit)}
	for {
		url := o.url + "/assets?" + queryParams.Encode()

		body, err := o.doOpenseaRequest(url)
		if err != nil {
			o.setConnected(false)
			return nil, err
		}

		container := AssetContainer{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			o.setConnected(false)
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

		if len(assets.Assets) >= limit {
			break
		}
	}

	o.setConnected(true)
	return assets, nil
}

func (o *Client) doOpenseaRequest(url string) ([]byte, error) {
	// Ensure only one thread makes a request at a time
	o.requestLock.Lock()
	defer o.requestLock.Unlock()

	retryCount := 0
	statusCode := http.StatusOK

	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0")
		if len(o.apiKey) > 0 {
			req.Header.Set("X-API-KEY", o.apiKey)
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
			if retryCount < RequestRetryMaxCount {
				// sleep and retry
				time.Sleep(RequestWaitTime)
				retryCount++
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
