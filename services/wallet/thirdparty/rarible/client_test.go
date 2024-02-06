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
				AnimationURL: "https://ipfs.raribleuserdata.com/ipfs/bafybeibpqyrvdkw7ypajsmsvjiz2mhytv7fyyfa6n35tfui7e473dxnyom/image.png",
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
				AnimationURL: "https://ipfs.raribleuserdata.com/ipfs/bafybeicsr36faeleunc5pzkqyf57pwm66vir4xhdkvv6cnkkznoyewqt7u/image.png",
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
