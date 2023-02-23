package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const AssetLimit = 50
const CollectionLimit = 300

var OpenseaClientInstances = make(map[uint64]*OpenseaClient)

var BaseURLs = map[uint64]string{
	1: "https://api.opensea.io/api/v1",
	4: "https://rinkeby-api.opensea.io/api/v1",
	5: "https://testnets-api.opensea.io/api/v1",
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

type OpenseaAssetContainer struct {
	Assets []OpenseaAsset `json:"assets"`
}

type OpenseaAssetCollection struct {
	Name string `json:"name"`
}

type OpenseaContract struct {
	Address string `json:"address"`
}

type OpenseaTrait struct {
	TraitType   string     `json:"trait_type"`
	Value       TraitValue `json:"value"`
	DisplayType string     `json:"display_type"`
	MaxValue    string     `json:"max_value"`
}

type OpenseaPaymentToken struct {
	ID       int    `json:"id"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	ImageURL string `json:"image_url"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	EthPrice string `json:"eth_price"`
	UsdPrice string `json:"usd_price"`
}

type OpenseaLastSale struct {
	PaymentToken OpenseaPaymentToken `json:"payment_token"`
}

type OpenseaSellOrder struct {
	CurrentPrice string `json:"current_price"`
}
type OpenseaAsset struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Permalink         string                 `json:"permalink"`
	ImageThumbnailURL string                 `json:"image_thumbnail_url"`
	ImageURL          string                 `json:"image_url"`
	Contract          OpenseaContract        `json:"asset_contract"`
	Collection        OpenseaAssetCollection `json:"collection"`
	Traits            []OpenseaTrait         `json:"traits"`
	LastSale          OpenseaLastSale        `json:"last_sale"`
	SellOrders        []OpenseaSellOrder     `json:"sell_orders"`
	BackgroundColor   string                 `json:"background_color"`
}

type OpenseaCollectionTrait struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type OpenseaCollection struct {
	Name            string                            `json:"name"`
	Slug            string                            `json:"slug"`
	ImageURL        string                            `json:"image_url"`
	OwnedAssetCount int                               `json:"owned_asset_count"`
	Traits          map[string]OpenseaCollectionTrait `json:"traits"`
}

type OpenseaClient struct {
	client          *http.Client
	url             string
	apiKey          string
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
}

// new opensea client.
func newOpenseaClient(chainID uint64, apiKey string) (*OpenseaClient, error) {
	if client, ok := OpenseaClientInstances[chainID]; ok {
		if client.apiKey == apiKey {
			return client, nil
		}
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}
	if url, ok := BaseURLs[chainID]; ok {
		openseaClient := &OpenseaClient{client: client, url: url, apiKey: apiKey, IsConnected: true, LastCheckedAt: time.Now().Unix()}
		OpenseaClientInstances[chainID] = openseaClient
		return openseaClient, nil
	}

	return nil, errors.New("ChainID not supported")
}

func (o *OpenseaClient) fetchAllCollectionsByOwner(owner common.Address) ([]OpenseaCollection, error) {
	offset := 0
	var collections []OpenseaCollection
	o.IsConnectedLock.Lock()
	defer o.IsConnectedLock.Unlock()
	o.LastCheckedAt = time.Now().Unix()
	for {
		url := fmt.Sprintf("%s/collections?asset_owner=%s&offset=%d&limit=%d", o.url, owner, offset, CollectionLimit)
		body, err := o.doOpenseaRequest(url)
		if err != nil {
			o.IsConnected = false
			return nil, err
		}

		var tmp []OpenseaCollection
		err = json.Unmarshal(body, &tmp)
		if err != nil {
			o.IsConnected = false
			return nil, err
		}

		collections = append(collections, tmp...)

		if len(tmp) < CollectionLimit {
			break
		}
	}
	o.IsConnected = true
	return collections, nil
}

func (o *OpenseaClient) fetchAllAssetsByOwnerAndCollection(owner common.Address, collectionSlug string, limit int) ([]OpenseaAsset, error) {
	offset := 0
	var assets []OpenseaAsset
	o.IsConnectedLock.Lock()
	defer o.IsConnectedLock.Unlock()
	o.LastCheckedAt = time.Now().Unix()
	for {
		url := fmt.Sprintf("%s/assets?owner=%s&collection=%s&offset=%d&limit=%d", o.url, owner, collectionSlug, offset, AssetLimit)
		body, err := o.doOpenseaRequest(url)
		if err != nil {
			o.IsConnected = false
			return nil, err
		}

		container := OpenseaAssetContainer{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			o.IsConnected = false
			return nil, err
		}

		for _, asset := range container.Assets {
			for i := range asset.Traits {
				asset.Traits[i].TraitType = strings.Replace(asset.Traits[i].TraitType, "_", " ", 1)
				asset.Traits[i].Value = TraitValue(strings.Title(string(asset.Traits[i].Value)))
			}
			assets = append(assets, asset)
		}

		if len(container.Assets) < AssetLimit {
			break
		}

		if len(assets) >= limit {
			break
		}
	}

	o.IsConnected = true
	return assets, nil
}

func (o *OpenseaClient) doOpenseaRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0")
	req.Header.Set("X-API-KEY", o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("failed to close opensea request body", "err", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}
