package wallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const AssetLimit = 50
const CollectionLimit = 300

type TraitValue string

func (st *TraitValue) UnmarshalJSON(b []byte) error {
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}
	switch v := item.(type) {
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
	client *http.Client
	url    string
}

// new opensea client.
func newOpenseaClient() *OpenseaClient {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	return &OpenseaClient{client: client, url: "https://api.opensea.io/api/v1"}
}

func (o *OpenseaClient) fetchAllCollectionsByOwner(owner common.Address) ([]OpenseaCollection, error) {
	offset := 0
	var collections []OpenseaCollection
	for {
		url := fmt.Sprintf("%s/collections?asset_owner=%s&offset=%d&limit=%d", o.url, owner, offset, CollectionLimit)
		body, err := o.doOpenseaRequest(url)
		if err != nil {
			return nil, err
		}

		var tmp []OpenseaCollection
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

func (o *OpenseaClient) fetchAllAssetsByOwnerAndCollection(owner common.Address, collectionSlug string, limit int) ([]OpenseaAsset, error) {
	offset := 0
	var assets []OpenseaAsset
	for {
		url := fmt.Sprintf("%s/assets?owner=%s&collection=%s&offset=%d&limit=%d", o.url, owner, collectionSlug, offset, AssetLimit)
		body, err := o.doOpenseaRequest(url)
		if err != nil {
			return nil, err
		}

		container := OpenseaAssetContainer{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			return nil, err
		}

		assets = append(assets, container.Assets...)

		if len(container.Assets) < AssetLimit {
			break
		}

		if len(assets) >= limit {
			break
		}
	}
	return assets, nil
}

func (o *OpenseaClient) doOpenseaRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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
