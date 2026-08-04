package rarible

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshallCollection(t *testing.T) {
	expectedCollectionData := thirdparty.CollectionData{
		ID: thirdparty.ContractID{
			ChainID: 1,
			Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
		},
		ContractType: w_common.ContractTypeERC721,
		Provider:     "rarible",
		Name:         "CryptoKitties",
		ImageURL:     "https://i.seadn.io/gae/C272ZRW1RGGef9vKMePFSCeKc1Lw6U40wl9ofNVxzUxFdj84hH9xJRQNf-7wgs7W8qw8RWe-1ybKp-VKuU5D-tg?w=500&auto=format",
		Traits:       make(map[string]thirdparty.CollectionTrait),
		Socials:      nil,
	}

	collection := Collection{}
	err := json.Unmarshal([]byte(collectionJSON), &collection)
	assert.NoError(t, err)

	contractID, err := raribleContractIDToUniqueID(collection.ID, true)
	assert.NoError(t, err)

	collectionData := collection.toCommon(contractID)
	assert.Equal(t, expectedCollectionData, collectionData)
}

func TestUnmarshallOwnedCollectibles(t *testing.T) {
	expectedTokenID0, _ := big.NewInt(0).SetString("32292934596187112148346015918544186536963932779440027682601542850818403729416", 10)
	expectedTokenID1, _ := big.NewInt(0).SetString("32292934596187112148346015918544186536963932779440027682601542850818403729414", 10)

	expectedCollectiblesData := []thirdparty.FullCollectibleData{
		{
			CollectibleData: thirdparty.CollectibleData{
				ID: thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						ChainID: 1,
						Address: common.HexToAddress("0xb66a603f4cfe17e3d27b87a8bfcad319856518b8"),
					},
					TokenID: &bigint.BigInt{
						Int: expectedTokenID0,
					},
				},
				ContractType: w_common.ContractTypeUnknown,
				Provider:     "rarible",
				Name:         "Rariversary #002",
				Description:  "Today marks your Second Rariversary! Can you believe it’s already been two years? Time flies when you’re having fun! Thank you for everything you contribute!",
				Permalink:    "https://rarible.com/token/0xb66a603f4cfe17e3d27b87a8bfcad319856518b8:32292934596187112148346015918544186536963932779440027682601542850818403729416",
				ImageURL:     "https://lh3.googleusercontent.com/03DCIWuHtWUG5zIPAkdBjPAucg-BNu-917hsY1LRyEtG9pMcYSwIv5n_jZoK4bvMjNbw9MEC3AZA29kje83fCf2XwG6WegOv0JU=s1000",
				ThumbnailURL: "https://lh3.googleusercontent.com/03DCIWuHtWUG5zIPAkdBjPAucg-BNu-917hsY1LRyEtG9pMcYSwIv5n_jZoK4bvMjNbw9MEC3AZA29kje83fCf2XwG6WegOv0JU=s250",
				Traits: []thirdparty.CollectibleTrait{
					{
						TraitType: "Theme",
						Value:     "Luv U",
					},
					{
						TraitType: "Gift for",
						Value:     "Rariversary",
					},
					{
						TraitType: "Year",
						Value:     "2",
					},
				},
				TokenURI: "ipfs://ipfs/bafkreialxjfvfkn43jluxmilfg3d3ojnomtqg634nuowqq2syx4odqrx5m",
			},
		},
		{
			CollectibleData: thirdparty.CollectibleData{
				ID: thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						ChainID: 1,
						Address: common.HexToAddress("0xb66a603f4cfe17e3d27b87a8bfcad319856518b8"),
					},
					TokenID: &bigint.BigInt{
						Int: expectedTokenID1,
					},
				},
				ContractType: w_common.ContractTypeUnknown,
				Provider:     "rarible",
				Name:         "Rariversary #003",
				Description:  "Today marks your Third Rariversary! Can you believe it’s already been three years? Time flies when you’re having fun! We’ve loved working with you these years and can’t wait to see what the next few years bring. Thank you for everything you contribute!",
				Permalink:    "https://rarible.com/token/0xb66a603f4cfe17e3d27b87a8bfcad319856518b8:32292934596187112148346015918544186536963932779440027682601542850818403729414",
				ImageURL:     "https://lh3.googleusercontent.com/SimzYIBjaTFt3BTBXFGOOvAqfw_etV0Pbe2pen-IvwF7L8DOysNca7qBdj3Dt5n_HWsse5vDLD7FZ7o5XdEivRvBtUybI1mXZEBQ=s1000",
				ThumbnailURL: "https://lh3.googleusercontent.com/SimzYIBjaTFt3BTBXFGOOvAqfw_etV0Pbe2pen-IvwF7L8DOysNca7qBdj3Dt5n_HWsse5vDLD7FZ7o5XdEivRvBtUybI1mXZEBQ=s250",
				Traits: []thirdparty.CollectibleTrait{
					{
						TraitType: "Theme",
						Value:     "LFG",
					},
					{
						TraitType: "Gift for",
						Value:     "Rariversary",
					},
					{
						TraitType: "Year",
						Value:     "3",
					},
				},
				TokenURI: "ipfs://ipfs/bafkreifeaueluerp33pjevz56f3ioxv63z73zuvm4wku5k6sobvala4phe",
			},
		},
	}

	var container CollectiblesContainer
	err := json.Unmarshal([]byte(ownedCollectiblesJSON), &container)
	assert.NoError(t, err)

	collectiblesData := raribleToCollectiblesData(container.Collectibles, true)

	assert.Equal(t, expectedCollectiblesData, collectiblesData)
}

func TestGetThumbnailURL(t *testing.T) {
	preview := Content{Type: "IMAGE", URL: "preview", Representation: "PREVIEW"}
	portrait := Content{Type: "IMAGE", URL: "portrait", Representation: "PORTRAIT"}
	big := Content{Type: "IMAGE", URL: "big", Representation: "BIG"}
	original := Content{Type: "IMAGE", URL: "original", Representation: "ORIGINAL"}
	video := Content{Type: "VIDEO", URL: "video", Representation: "PREVIEW"}

	testCases := []struct {
		name     string
		contents []Content
		expected string
	}{
		{
			name:     "prefers the preview over every larger representation",
			contents: []Content{original, big, portrait, preview},
			expected: "preview",
		},
		{
			name:     "falls back to the portrait when there is no preview",
			contents: []Content{original, big, portrait},
			expected: "portrait",
		},
		{
			name:     "stays empty rather than offering the original as a thumbnail",
			contents: []Content{original},
			expected: "",
		},
		{
			name:     "stays empty rather than offering the big representation",
			contents: []Content{original, big},
			expected: "",
		},
		{
			name:     "ignores video content",
			contents: []Content{video, original},
			expected: "",
		},
		{
			name:     "handles an item with no content at all",
			contents: []Content{},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getThumbnailURL(tc.contents))
		})
	}
}

func TestGetAnimation(t *testing.T) {
	video := Content{Type: "VIDEO", URL: "video", Representation: "ORIGINAL", MimeType: "video/mp4"}
	videoNoMime := Content{Type: "VIDEO", URL: "video", Representation: "ORIGINAL"}
	gif := Content{Type: "IMAGE", URL: "gif", Representation: "ORIGINAL", MimeType: "image/gif"}
	still := Content{Type: "IMAGE", URL: "still", Representation: "ORIGINAL", MimeType: "image/png"}
	stillNoMime := Content{Type: "IMAGE", URL: "still", Representation: "ORIGINAL"}

	testCases := []struct {
		name              string
		contents          []Content
		expectedURL       string
		expectedMediaType string
	}{
		{
			name:              "carries the mime type of the video it picked",
			contents:          []Content{still, video},
			expectedURL:       "video",
			expectedMediaType: "video/mp4",
		},
		{
			name:              "carries the mime type of an animated image",
			contents:          []Content{still, gif},
			expectedURL:       "gif",
			expectedMediaType: "image/gif",
		},
		{
			name:              "prefers video over an animated image",
			contents:          []Content{gif, video},
			expectedURL:       "video",
			expectedMediaType: "video/mp4",
		},
		{
			name:              "reports no animation for a still image",
			contents:          []Content{still},
			expectedURL:       "",
			expectedMediaType: "",
		},
		{
			name:              "reports no animation when the mime type is unknown",
			contents:          []Content{stillNoMime},
			expectedURL:       "",
			expectedMediaType: "",
		},
		{
			// The consumer falls back to resolving the media type itself, so the
			// video is still offered rather than dropped.
			name:              "offers a video whose mime type the provider omitted",
			contents:          []Content{videoNoMime},
			expectedURL:       "video",
			expectedMediaType: "",
		},
		{
			name:              "handles an item with no content at all",
			contents:          []Content{},
			expectedURL:       "",
			expectedMediaType: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			animation := getAnimation(tc.contents)
			assert.Equal(t, tc.expectedURL, animation.URL)
			assert.Equal(t, tc.expectedMediaType, animation.MimeType)
		})
	}
}
