package opensea

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshallDetailedAsset(t *testing.T) {
	nftJSON := `{"nft": {"identifier": "7", "collection": "test-cool-cats-v3", "contract": "0x9a95631794a42d30c47f214fbe02a72585df35e1", "token_standard": "erc721", "name": "Cool Cat #7", "description": "Cool Cats also live on Rinkeby for testing purposes!", "image_url": "https://i.seadn.io/gae/blGPn5DZZXjGkrwx5EICajgepVjogYxNEnvDWBECCiFy7OXJkaW--d9OgO4gXqZzkFhd87pd_ckOyS-nEGLVymPjccavrznJGJ4XgA?w=500&auto=format", "metadata_url": "https://BetaTestPetMetadataServer.grampabacon.repl.co/cat/7", "created_at": "2022-10-05T14:29:31.966382", "updated_at": "2022-10-05T14:29:43.889730", "is_disabled": false, "is_nsfw": false, "is_suspicious": false, "creator": "0x772b92a6abe5129f8ef91d164cc757dd9bbd0bc7", "traits": [{"trait_type": "tier", "display_type": null, "max_value": null, "trait_count": 0, "order": null, "value": "cool_2"}], "owners": [{"address": "0x0000000000000000000000000000000000000001", "quantity": 1}], "rarity": null}}`
	expectedNFT := DetailedNFT{
		TokenID:       &bigint.BigInt{Int: big.NewInt(7)},
		Collection:    "test-cool-cats-v3",
		Contract:      common.HexToAddress("0x9a95631794a42d30c47f214fbe02a72585df35e1"),
		TokenStandard: "erc721",
		Name:          "Cool Cat #7",
		Description:   "Cool Cats also live on Rinkeby for testing purposes!",
		ImageURL:      "https://i.seadn.io/gae/blGPn5DZZXjGkrwx5EICajgepVjogYxNEnvDWBECCiFy7OXJkaW--d9OgO4gXqZzkFhd87pd_ckOyS-nEGLVymPjccavrznJGJ4XgA?w=500&auto=format",
		MetadataURL:   "https://BetaTestPetMetadataServer.grampabacon.repl.co/cat/7",
		Owners: []OwnerV2{
			{
				Address:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
				Quantity: &bigint.BigInt{Int: big.NewInt(1)},
			},
		},
		Traits: []TraitV2{
			{
				TraitType:   "tier",
				DisplayType: "",
				MaxValue:    "",
				TraitCount:  0,
				Order:       "",
				Value:       "cool_2",
			},
		},
	}

	nftContainer := DetailedNFTContainer{}
	err := json.Unmarshal([]byte(nftJSON), &nftContainer)
	assert.NoError(t, err)
	assert.Equal(t, expectedNFT, nftContainer.NFT)
}
