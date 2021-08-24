package wallet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
)

func TestFetchAllCollectionsByOwner(t *testing.T) {
	expected := []OpenseaCollection{OpenseaCollection{Name: "Rocky", Slug: "rocky", ImageURL: "ImageUrl", OwnedAssetCount: 1}}
	response, _ := json.Marshal(expected)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := &OpenseaClient{
		client:           srv.Client(),
		url:              srv.URL,
		collectionsCache: make(map[common.Address][]OpenseaCollection),
		assetsCache:      make(map[common.Address]map[string][]OpenseaAsset),
	}
	res, err := opensea.fetchAllCollectionsByOwner(common.Address{1})
	assert.Equal(t, expected, res)
	assert.Nil(t, err)
}

func TestFetchAllAssetsByOwnerAndCollection(t *testing.T) {
	expected := []OpenseaAsset{OpenseaAsset{
		ID:                1,
		Name:              "Rocky",
		Description:       "Rocky Balboa",
		Permalink:         "permalink",
		ImageThumbnailURL: "ImageThumbnailURL",
		ImageURL:          "ImageUrl",
		Contract:          OpenseaContract{Address: "1"},
		Collection:        OpenseaAssetCollection{Name: "Rocky"},
	}}
	response, _ := json.Marshal(OpenseaAssetContainer{Assets: expected})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := &OpenseaClient{
		client:           srv.Client(),
		url:              srv.URL,
		collectionsCache: make(map[common.Address][]OpenseaCollection),
		assetsCache:      make(map[common.Address]map[string][]OpenseaAsset),
	}
	res, err := opensea.fetchAllAssetsByOwnerAndCollection(common.Address{1}, "rocky", 200)
	assert.Equal(t, expected, res)
	assert.Nil(t, err)
}
