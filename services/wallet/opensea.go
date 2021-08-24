package wallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const AssetLimit = 50
const CollectionLimit = 300

type OpenseaAssetContainer struct {
	Assets []OpenseaAsset `json:"assets"`
}

type OpenseaAssetCollection struct {
	Name string `json:"name"`
}

type OpenseaContract struct {
	Address string `json:"address"`
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
}

type OpenseaCollection struct {
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	ImageURL        string `json:"image_url"`
	OwnedAssetCount int    `json:"owned_asset_count"`
}

type OpenseaClient struct {
	client           *http.Client
	url              string
	collectionsCache map[common.Address][]OpenseaCollection
	assetsCache      map[common.Address]map[string][]OpenseaAsset
}

// new opensea client.
func newOpenseaClient() *OpenseaClient {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	return &OpenseaClient{
		client:           client,
		url:              "https://api.opensea.io/api/v1",
		collectionsCache: make(map[common.Address][]OpenseaCollection),
		assetsCache:      make(map[common.Address]map[string][]OpenseaAsset),
	}
}

func (o *OpenseaClient) fetchAllCollectionsByOwner(owner common.Address) ([]OpenseaCollection, error) {
	if cachedCollections, ok := o.collectionsCache[owner]; ok {
		return cachedCollections, nil
	}

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
	o.collectionsCache[owner] = collections
	return collections, nil
}

func (o *OpenseaClient) fetchAllAssetsByOwnerAndCollection(owner common.Address, collectionSlug string, limit int) ([]OpenseaAsset, error) {
	if _, ok := o.assetsCache[owner]; !ok {
		o.assetsCache[owner] = make(map[string][]OpenseaAsset)
	}

	if cachedAssets, ok := o.assetsCache[owner][collectionSlug]; ok {
		return cachedAssets, nil
	}

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

	o.assetsCache[owner][collectionSlug] = assets
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
