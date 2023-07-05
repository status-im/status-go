package thirdparty

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type CollectibleUniqueID struct {
	ContractAddress common.Address `json:"contractAddress"`
	TokenID         *bigint.BigInt `json:"tokenID"`
}

func (k *CollectibleUniqueID) HashKey() string {
	return k.ContractAddress.String() + "+" + k.TokenID.String()
}

type CollectibleMetadata struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	CollectionImageURL string `json:"collection_image"`
	ImageURL           string `json:"image"`
}

type CollectibleMetadataProvider interface {
	CanProvideCollectibleMetadata(chainID uint64, id CollectibleUniqueID, tokenURI string) (bool, error)
	FetchCollectibleMetadata(chainID uint64, id CollectibleUniqueID, tokenURI string) (*CollectibleMetadata, error)
}

type TokenBalance struct {
	TokenID *bigint.BigInt `json:"tokenId"`
	Balance *bigint.BigInt `json:"balance"`
}

type TokenBalancesPerContractAddress = map[common.Address][]TokenBalance

type CollectibleOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type CollectibleContractOwnership struct {
	ContractAddress common.Address     `json:"contractAddress"`
	Owners          []CollectibleOwner `json:"owners"`
}

type CollectibleContractOwnershipProvider interface {
	FetchCollectibleOwnersByContractAddress(chainID uint64, contractAddress common.Address) (*CollectibleContractOwnership, error)
	IsChainSupported(chainID uint64) bool
}
