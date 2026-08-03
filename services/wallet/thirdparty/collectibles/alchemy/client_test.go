package alchemy

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientWithParams_ProxyBasicWithCustomClient(t *testing.T) {
	c := NewClientWithParams(Params{
		IsProxy:    true,
		HttpClient: &http.Client{Transport: http.DefaultTransport},
		Creds: &thirdparty.BasicCreds{
			User:     security.NewSensitiveString("u"),
			Password: security.NewSensitiveString("p"),
		},
	})
	require.Equal(t, thirdparty.AuthTypeBasic, c.authTransport.Auth().Type)
}

func TestUnmarshallCollection(t *testing.T) {
	expectedCollectionData := thirdparty.CollectionData{
		ID: thirdparty.ContractID{
			ChainID: 1,
			Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
		},
		ContractType: w_common.ContractTypeERC721,
		Provider:     "alchemy",
		Name:         "CryptoKitties",
		ImageURL:     "https://i.seadn.io/gae/C272ZRW1RGGef9vKMePFSCeKc1Lw6U40wl9ofNVxzUxFdj84hH9xJRQNf-7wgs7W8qw8RWe-1ybKp-VKuU5D-tg?w=500&auto=format",
		Traits:       make(map[string]thirdparty.CollectionTrait),
		Socials:      &thirdparty.CollectionSocials{Website: "", TwitterHandle: "CryptoKitties", Provider: "alchemy"},
	}

	collection := Contract{}
	err := json.Unmarshal([]byte(collectionJSON), &collection)
	assert.NoError(t, err)

	contractID := thirdparty.ContractID{
		ChainID: 1,
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}

	collectionData := collection.toCollectionData(contractID)
	assert.Equal(t, expectedCollectionData, collectionData)
}

func TestUnmarshallOwnedCollectibles(t *testing.T) {
	owner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	expectedTokenID0, _ := big.NewInt(0).SetString("50659039041325838222074459099120411190538227963344971355684955900852972814336", 10)
	expectedTokenID1, _ := big.NewInt(0).SetString("900", 10)

	expectedBalance0, _ := big.NewInt(0).SetString("15", 10)
	expectedBalance1, _ := big.NewInt(0).SetString("1", 10)

	expectedCollectiblesData := []thirdparty.FullCollectibleData{
		{
			CollectibleData: thirdparty.CollectibleData{
				ID: thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						ChainID: 1,
						Address: common.HexToAddress("0x2b1870752208935fDA32AB6A016C01a27877CF12"),
					},
					TokenID: &bigint.BigInt{
						Int: expectedTokenID0,
					},
				},
				ContractType: w_common.ContractTypeERC1155,
				Provider:     "alchemy",
				Name:         "HODL",
				Description:  "The enemy king sent a single message, written on a parchment stained by blood.\n“You are advised to submit without further delay, for if I bring my army into your land, I will destroy your hodlings, slay your people, and burn your city to ashes.”\nHodlers of ENJ sent a single word as reply:\n“If.”\nThe battle that followed does not come around too often, a battle that began every legend told about the warriors that gained eternal glory. \nThe battle that followed seemed like a lost one from the very beginning. \nThe enemy army was revealed at dawn, illuminated by the rising Sun.The ground shook as countless hordes marched towards a small band of men armed with shields, spears and swords.\nThe hodlers were outnumbered, one thousand to one. \nFear, doubt and uncertainty did not reach their hearts and minds - for they were born for this. \nEach hodler was bred for warfare, instructed in bloodshed, groomed to become a poet of death. \nA philosopher of war, blood and glory. \nEach man was forged into an invincible soldier that had a single driving force during each battle.\nStand your ground - at all costs. \nAs the swarm of enemies approached, the king yelled, asking his men: \n“Hodlers! What is your profession?”\n“HODL! HODL! HODL! HODL!!! HODL!!!!!” they replied, hitting spears against their shields. \nAn endless stream of arrows fell from the heavens only moments later, blocking out the Sun so they could fight in the shade. They emerged from the darkness without even a single scratch, protected by their legendary Enjin shields. \nWave after wave, their enemies rushed towards their doom, as they were met with cold tips of thrusting spears and sharp edges of crimson swords.\nAgainst all odds, the wall of men and steel held against the never-ending, shilling swarm. \nWhat was left of the enemy army retreated, fleeing in absolute panic and indisputable terror.\nBathed in blood, the ENJ hodlers were victorious.\nTheir story will be told for thousands of years, immortalized with divine blocks and chains.\n* * *\n“HODL” was minted in 2018 for our amazing community of epic Enjin HODLers. We are extremely grateful for the trust you've put in us and the products we're making - and the mission we're trying to accomplish, and hope you’ll love this token of our appreciation. ", // nolint: misspell
				Permalink:    "",
				ImageURL:     "https://res.cloudinary.com/alchemyapi/image/upload/convert-png/eth-mainnet/c5c93ffa8146ade7d3694c0f28463f0c",
				ThumbnailURL: "https://res.cloudinary.com/alchemyapi/image/upload/thumbnailv2/eth-mainnet/c5c93ffa8146ade7d3694c0f28463f0c",
				Traits:       []thirdparty.CollectibleTrait{},
				TokenURI:     "https://cdn.enjin.io/mint/meta/70000000000001b2.json",
			},
			CollectionData: &thirdparty.CollectionData{
				ID: thirdparty.ContractID{
					ChainID: 1,
					Address: common.HexToAddress("0x2b1870752208935fDA32AB6A016C01a27877CF12"),
				},
				ContractType: w_common.ContractTypeERC1155,
				Provider:     "alchemy",
				Name:         "",
				Slug:         "",
				ImageURL:     "",
				Traits:       make(map[string]thirdparty.CollectionTrait),
				Socials:      &thirdparty.CollectionSocials{Website: "", TwitterHandle: "", Provider: "alchemy"},
			},
			Ownership: []thirdparty.AccountBalance{
				{
					Address:     owner,
					Balance:     &bigint.BigInt{Int: expectedBalance0},
					TxTimestamp: -1,
				},
			},
		},
		{
			CollectibleData: thirdparty.CollectibleData{
				ID: thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						ChainID: 1,
						Address: common.HexToAddress("0x3f6B1585AfeFc56433C8d28AA89dbc77af59278f"),
					},
					TokenID: &bigint.BigInt{
						Int: expectedTokenID1,
					},
				},
				ContractType: w_common.ContractTypeERC721,
				Provider:     "alchemy",
				Name:         "#900",
				Description:  "5,555 SimpsonPunks entered the Ethereum Blockchain🍩",
				Permalink:    "",
				ImageURL:     "https://res.cloudinary.com/alchemyapi/image/upload/convert-png/eth-mainnet/52accf48dc609088738b15808fe07e8c",
				ThumbnailURL: "https://res.cloudinary.com/alchemyapi/image/upload/thumbnailv2/eth-mainnet/52accf48dc609088738b15808fe07e8c",
				Traits: []thirdparty.CollectibleTrait{
					{
						TraitType: "layers",
						Value:     "Background",
					},
					{
						TraitType: "Face",
						Value:     "Monkey",
					},
					{
						TraitType: "Head",
						Value:     "Sweatband Blue",
					},
					{
						TraitType: "Facial Hair",
						Value:     "Thin Full",
					},
					{
						TraitType: "Mouth",
						Value:     "Burger",
					},
				},
				TokenURI: "https://alchemy.mypinata.cloud/ipfs/bafybeidqbmbglapk2bkffa4o2ws5jhxnhlbdeqh7k6tk62pukse3xhvv2e/900.json",
			},
			CollectionData: &thirdparty.CollectionData{
				ID: thirdparty.ContractID{
					ChainID: 1,
					Address: common.HexToAddress("0x3f6B1585AfeFc56433C8d28AA89dbc77af59278f"),
				},
				ContractType: w_common.ContractTypeERC721,
				Provider:     "alchemy",
				Name:         "Simpson Punk",
				Slug:         "",
				ImageURL:     "https://raw.seadn.io/files/e7765f13c4658f514d0efc008ae7f300.png",
				Traits:       make(map[string]thirdparty.CollectionTrait),
				Socials:      &thirdparty.CollectionSocials{Website: "", TwitterHandle: "SimpsonPunksETH", Provider: "alchemy"},
			},
			Ownership: []thirdparty.AccountBalance{
				{
					Address:     owner,
					Balance:     &bigint.BigInt{Int: expectedBalance1},
					TxTimestamp: -1,
				},
			},
		},
	}

	var container OwnedNFTList
	err := json.Unmarshal([]byte(ownedCollectiblesJSON), &container)
	assert.NoError(t, err)

	collectiblesData := alchemyToCollectiblesData(w_common.ChainID(w_common.EthereumMainnet), container.OwnedNFTs, &owner)

	assert.Equal(t, expectedCollectiblesData, collectiblesData)
}

type testURLResolver struct {
	baseURL string
}

func (r testURLResolver) GetNFTBaseURL(chainID w_common.ChainID) (string, error) {
	return r.baseURL, nil
}

func (r testURLResolver) IsChainSupported(chainID w_common.ChainID) bool {
	return true
}

func makeNFTResponse(t *testing.T, w http.ResponseWriter, nftCount int, nextPageKey string) {
	t.Helper()

	nfts := make([]map[string]any, 0, nftCount)
	for i := range nftCount {
		nfts = append(nfts, map[string]any{
			"contract": map[string]any{
				"address":   "0x0000000000000000000000000000000000000001",
				"tokenType": "ERC721",
			},
			"tokenId": fmt.Sprintf("%d", i+1),
			"name":    fmt.Sprintf("NFT #%d", i+1),
			"raw": map[string]any{
				"tokenUri":    "",
				"metadata":    map[string]any{},
				"error":       nil,
				"rawMetadata": map[string]any{"attributes": []any{}},
			},
			"image": map[string]any{},
		})
	}

	resp := map[string]any{
		"ownedNfts": nfts,
	}
	if nextPageKey != "" {
		resp["pageKey"] = nextPageKey
	}

	require.NoError(t, json.NewEncoder(w).Encode(resp))
}

func newTestClient(baseURL string) *Client {
	client := NewClient(security.NewSensitiveString(""))
	client.urlResolver = testURLResolver{baseURL: baseURL}
	return client
}

var testOwner = common.HexToAddress("0x1234567890123456789012345678901234567890")

func TestFetchOwnedAssetsSendsPageSizeAndExcludeFilters(t *testing.T) {
	var called atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		assert.Equal(t, "/getNFTsForOwner", r.URL.Path)
		assert.Equal(t, "500", r.URL.Query().Get("pageSize"))
		assert.Equal(t, "SPAM", r.URL.Query().Get("excludeFilters"))
		makeNFTResponse(t, w, 1, "")
	}))
	defer server.Close()

	assets, err := newTestClient(server.URL).FetchAllAssetsByOwner(
		context.Background(), w_common.ChainID(w_common.EthereumMainnet), testOwner, "", thirdparty.FetchNoLimit,
	)
	require.NoError(t, err)
	require.NotNil(t, assets)
	assert.True(t, called.Load())
	assert.Len(t, assets.Items, 1)
	assert.Empty(t, assets.NextCursor)
}

func TestFetchOwnedAssetsPagination(t *testing.T) {
	tests := []struct {
		name            string
		limit           int
		handler         func(t *testing.T, page int32, w http.ResponseWriter, r *http.Request)
		expectedPages   int32
		expectedItems   int
		expectCursorSet bool
	}{
		{
			name:  "stops on empty page key",
			limit: thirdparty.FetchNoLimit,
			handler: func(t *testing.T, page int32, w http.ResponseWriter, r *http.Request) {
				if page < 3 {
					makeNFTResponse(t, w, 2, fmt.Sprintf("page-%d", page))
				} else {
					makeNFTResponse(t, w, 1, "")
				}
			},
			expectedPages:   3,
			expectedItems:   5,
			expectCursorSet: false,
		},
		{
			name:  "stops at page cap for unbounded fetch",
			limit: thirdparty.FetchNoLimit,
			handler: func(t *testing.T, page int32, w http.ResponseWriter, r *http.Request) {
				makeNFTResponse(t, w, 1, fmt.Sprintf("page-%d", page))
			},
			expectedPages:   int32(fetchNoLimitMaxPages),
			expectedItems:   fetchNoLimitMaxPages,
			expectCursorSet: true,
		},
		{
			name:  "honors limit before page cap",
			limit: 3,
			handler: func(t *testing.T, page int32, w http.ResponseWriter, r *http.Request) {
				makeNFTResponse(t, w, 1, fmt.Sprintf("page-%d", page))
			},
			expectedPages:   3,
			expectedItems:   3,
			expectCursorSet: true,
		},
		{
			name:  "trims items to limit",
			limit: 3,
			handler: func(t *testing.T, _ int32, w http.ResponseWriter, r *http.Request) {
				makeNFTResponse(t, w, 5, "next-page")
			},
			expectedPages:   1,
			expectedItems:   3,
			expectCursorSet: true,
		},
		{
			name:  "single page no next cursor",
			limit: thirdparty.FetchNoLimit,
			handler: func(t *testing.T, _ int32, w http.ResponseWriter, r *http.Request) {
				makeNFTResponse(t, w, 2, "")
			},
			expectedPages:   1,
			expectedItems:   2,
			expectCursorSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := requestCount.Add(1)
				if page == 1 {
					assert.Empty(t, r.URL.Query().Get("pageKey"))
				} else {
					assert.NotEmpty(t, r.URL.Query().Get("pageKey"))
				}
				tt.handler(t, page, w, r)
			}))
			defer server.Close()

			assets, err := newTestClient(server.URL).FetchAllAssetsByOwner(
				context.Background(), w_common.ChainID(w_common.EthereumMainnet), testOwner, "", tt.limit,
			)
			require.NoError(t, err)
			require.NotNil(t, assets)
			assert.Equal(t, tt.expectedPages, requestCount.Load())
			assert.Len(t, assets.Items, tt.expectedItems)
			if tt.expectCursorSet {
				assert.NotEmpty(t, assets.NextCursor)
			} else {
				assert.Empty(t, assets.NextCursor)
			}
		})
	}
}

func TestFetchOwnedAssetsRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		makeNFTResponse(t, w, 1, "next")
	}))
	defer server.Close()

	_, err := newTestClient(server.URL).FetchAllAssetsByOwner(
		ctx, w_common.ChainID(w_common.EthereumMainnet), testOwner, "", thirdparty.FetchNoLimit,
	)
	require.Error(t, err)
}
