package opensea

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
)

const (
	ExpiredKeyError         = "Expired API key"
	ExpectedExpiredKeyError = "invalid json: Expired API key"
)

func initTestClient(srv *httptest.Server) *Client {
	urlGetter := func(chainID walletCommon.ChainID, path string) (string, error) {
		return srv.URL, nil
	}

	status := connection.NewStatus("", nil)

	client := &HTTPClient{
		client: srv.Client(),
	}
	opensea := &Client{
		client:           client,
		connectionStatus: status,
		urlGetter:        urlGetter,
	}

	return opensea
}

func TestFetchAllCollectionsByOwner(t *testing.T) {
	expectedOS := []OwnedCollection{{
		Collection: Collection{
			Name:     "Rocky",
			Slug:     "rocky",
			ImageURL: "ImageUrl",
		},
		OwnedAssetCount: &bigint.BigInt{Int: big.NewInt(1)},
	}}
	response, _ := json.Marshal(expectedOS)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := initTestClient(srv)
	res, err := opensea.FetchAllCollectionsByOwner(walletCommon.ChainID(1), common.Address{1})
	assert.Equal(t, expectedOS, res)
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

	opensea := initTestClient(srv)
	res, err := opensea.FetchAllCollectionsByOwner(walletCommon.ChainID(1), common.Address{1})
	assert.Nil(t, res)
	assert.Equal(t, err, fmt.Errorf(ExpectedExpiredKeyError))
}

func TestFetchAllAssetsByOwnerAndCollection(t *testing.T) {
	expectedOS := AssetContainer{
		Assets: []Asset{{
			ID:                1,
			TokenID:           &bigint.BigInt{Int: big.NewInt(1)},
			Name:              "Rocky",
			Description:       "Rocky Balboa",
			Permalink:         "permalink",
			ImageThumbnailURL: "ImageThumbnailURL",
			ImageURL:          "ImageUrl",
			Contract: Contract{
				Address:         "1",
				ChainIdentifier: "ethereum",
			},
			Collection: Collection{
				Name:   "Rocky",
				Traits: map[string]CollectionTrait{},
			},
			Traits: []Trait{},
		}},
		NextCursor:     "",
		PreviousCursor: "",
	}
	expectedCommon := thirdparty.FullCollectibleDataContainer{
		Items: []thirdparty.FullCollectibleData{
			thirdparty.FullCollectibleData{
				CollectibleData: thirdparty.CollectibleData{
					ID: thirdparty.CollectibleUniqueID{
						ContractID: thirdparty.ContractID{
							ChainID: 1,
							Address: common.HexToAddress("0x1"),
						},
						TokenID: &bigint.BigInt{Int: big.NewInt(1)},
					},
					Name:        "Rocky",
					Description: "Rocky Balboa",
					Permalink:   "permalink",
					ImageURL:    "ImageUrl",
					Traits:      []thirdparty.CollectibleTrait{},
				},
				CollectionData: &thirdparty.CollectionData{
					ID: thirdparty.ContractID{
						ChainID: 1,
						Address: common.HexToAddress("0x1"),
					},
					Name:   "Rocky",
					Traits: map[string]thirdparty.CollectionTrait{},
				},
			},
		},
		NextCursor:     "",
		PreviousCursor: "",
	}
	response, _ := json.Marshal(expectedOS)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	opensea := initTestClient(srv)
	res, err := opensea.FetchAllAssetsByOwnerAndCollection(walletCommon.ChainID(1), common.Address{1}, "rocky", "", 200)
	assert.Nil(t, err)
	assert.Equal(t, expectedCommon, *res)
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

	opensea := initTestClient(srv)
	res, err := opensea.FetchAllAssetsByOwnerAndCollection(walletCommon.ChainID(1), common.Address{1}, "rocky", "", 200)
	assert.Nil(t, res)
	assert.Equal(t, fmt.Errorf(ExpectedExpiredKeyError), err)
}
