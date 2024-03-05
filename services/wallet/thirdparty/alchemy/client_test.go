package alchemy

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
		Provider:     "alchemy",
		Name:         "CryptoKitties",
		ImageURL:     "https://i.seadn.io/gae/C272ZRW1RGGef9vKMePFSCeKc1Lw6U40wl9ofNVxzUxFdj84hH9xJRQNf-7wgs7W8qw8RWe-1ybKp-VKuU5D-tg?w=500&auto=format",
		Traits:       make(map[string]thirdparty.CollectionTrait),
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
				AnimationURL: "https://nft-cdn.alchemy.com/eth-mainnet/c5c93ffa8146ade7d3694c0f28463f0c",
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
			},
			AccountBalance: &bigint.BigInt{
				Int: expectedBalance0,
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
				AnimationURL: "https://nft-cdn.alchemy.com/eth-mainnet/52accf48dc609088738b15808fe07e8c",
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
			},
			AccountBalance: &bigint.BigInt{
				Int: expectedBalance1,
			},
		},
	}

	var container OwnedNFTList
	err := json.Unmarshal([]byte(ownedCollectiblesJSON), &container)
	assert.NoError(t, err)

	collectiblesData := alchemyToCollectiblesData(w_common.ChainID(w_common.EthereumMainnet), container.OwnedNFTs)

	assert.Equal(t, expectedCollectiblesData, collectiblesData)
}
