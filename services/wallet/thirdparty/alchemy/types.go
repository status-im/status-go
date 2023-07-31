package alchemy

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type TokenBalance struct {
	TokenID *bigint.HexBigInt `json:"tokenId"`
	Balance *bigint.BigInt    `json:"balance"`
}

type CollectibleOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type CollectibleContractOwnership struct {
	Owners  []CollectibleOwner `json:"ownerAddresses"`
	PageKey string             `json:"pageKey"`
}

func alchemyOwnershipToCommon(contractAddress common.Address, alchemyOwnership CollectibleContractOwnership) (*thirdparty.CollectibleContractOwnership, error) {
	owners := make([]thirdparty.CollectibleOwner, 0, len(alchemyOwnership.Owners))
	for _, alchemyOwner := range alchemyOwnership.Owners {
		balances := make([]thirdparty.TokenBalance, 0, len(alchemyOwner.TokenBalances))

		for _, alchemyBalance := range alchemyOwner.TokenBalances {
			balances = append(balances, thirdparty.TokenBalance{
				TokenID: &bigint.BigInt{Int: alchemyBalance.TokenID.Int},
				Balance: alchemyBalance.Balance,
			})
		}
		owner := thirdparty.CollectibleOwner{
			OwnerAddress:  alchemyOwner.OwnerAddress,
			TokenBalances: balances,
		}

		owners = append(owners, owner)
	}

	ownership := thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          owners,
	}

	return &ownership, nil
}
