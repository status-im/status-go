package infura

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type CollectibleOwner struct {
	ContractAddress common.Address `json:"tokenAddress"`
	TokenID         *bigint.BigInt `json:"tokenId"`
	Amount          *bigint.BigInt `json:"amount"`
	OwnerAddress    common.Address `json:"ownerOf"`
}

type CollectibleContractOwnership struct {
	Owners  []CollectibleOwner `json:"owners"`
	Network string             `json:"network"`
	Cursor  string             `json:"cursor"`
}

func infuraOwnershipToCommon(contractAddress common.Address, ownersMap map[common.Address][]CollectibleOwner) (*thirdparty.CollectibleContractOwnership, error) {
	owners := make([]thirdparty.CollectibleOwner, 0, len(ownersMap))

	for ownerAddress, ownerTokens := range ownersMap {
		tokenBalances := make([]thirdparty.TokenBalance, 0, len(ownerTokens))

		for _, token := range ownerTokens {
			tokenBalances = append(tokenBalances, thirdparty.TokenBalance{
				TokenID: token.TokenID,
				Balance: token.Amount,
			})
		}

		owners = append(owners, thirdparty.CollectibleOwner{
			OwnerAddress:  ownerAddress,
			TokenBalances: tokenBalances,
		})
	}

	ownership := thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          owners,
	}

	return &ownership, nil
}
