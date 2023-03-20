package opensea

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/status-im/status-go/services/wallet/bigint"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
)

const (
	ExpiredKeyError         = "Expired API key"
	ExpectedExpiredKeyError = "invalid json: Expired API key"
)

func TestFetchAllCollectionsByOwner(t *testing.T) {
	expected := []OwnedCollection{{
		Collection: Collection{
			Name:     "Rocky",
			Slug:     "rocky",
			ImageURL: "ImageUrl",
		},
		OwnedAssetCount: &bigint.BigInt{Int: big.NewInt(1)},
	}}
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

func TestFetchAllCollectionsByOwnerWithInValidJson(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte(ExpiredKeyError))
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
	assert.Nil(t, res)
	assert.Equal(t, err, fmt.Errorf(ExpectedExpiredKeyError))
}

func TestFetchAllAssetsByOwnerAndCollection(t *testing.T) {
	expected := AssetContainer{
		Assets: []Asset{{
			ID:                1,
			Name:              "Rocky",
			Description:       "Rocky Balboa",
			Permalink:         "permalink",
			ImageThumbnailURL: "ImageThumbnailURL",
			ImageURL:          "ImageUrl",
			Contract:          Contract{Address: "1"},
			Collection:        Collection{Name: "Rocky"},
		}},
		NextCursor:     "",
		PreviousCursor: "",
	}
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
	res, err := opensea.FetchAllAssetsByOwnerAndCollection(common.Address{1}, "rocky", "", 200)
	assert.Nil(t, err)
	assert.Equal(t, expected, *res)
}

func TestFetchAllAssetsByOwnerAndCollectionInvalidJson(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte(ExpiredKeyError))
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := &Client{
		client: srv.Client(),
		url:    srv.URL,
	}
	res, err := opensea.FetchAllAssetsByOwnerAndCollection(common.Address{1}, "rocky", "", 200)
	assert.Nil(t, res)
	assert.Equal(t, fmt.Errorf(ExpectedExpiredKeyError), err)
}
