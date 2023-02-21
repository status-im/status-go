package opensea

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
)

func TestFetchAllCollectionsByOwner(t *testing.T) {
	expected := []Collection{Collection{Name: "Rocky", Slug: "rocky", ImageURL: "ImageUrl", OwnedAssetCount: 1}}
	response, _ := json.Marshal(expected)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := &Client{
		client: srv.Client(),
		url:    srv.URL,
	}
	res, err := opensea.FetchAllCollectionsByOwner(common.Address{1})
	assert.Equal(t, expected, res)
	assert.Nil(t, err)
}

func TestFetchAllAssetsByOwnerAndCollection(t *testing.T) {
	expected := []Asset{Asset{
		ID:                1,
		Name:              "Rocky",
		Description:       "Rocky Balboa",
		Permalink:         "permalink",
		ImageThumbnailURL: "ImageThumbnailURL",
		ImageURL:          "ImageUrl",
		Contract:          Contract{Address: "1"},
		Collection:        AssetCollection{Name: "Rocky"},
	}}
	response, _ := json.Marshal(AssetContainer{Assets: expected})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := &Client{
		client: srv.Client(),
		url:    srv.URL,
	}
	res, err := opensea.FetchAllAssetsByOwnerAndCollection(common.Address{1}, "rocky", 200)
	assert.Equal(t, expected, res)
	assert.Nil(t, err)
}
