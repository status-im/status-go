package wallet

import (
	"github.com/ethereum/go-ethereum/common"
	collectibles "github.com/status-im/status-go/services/wallet/collectibles"
	collectibles_ownership "github.com/status-im/status-go/services/wallet/collectibles/ownership"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
)

type MultistandardBalanceTokenListProvider struct {
	tokenManager *token.Manager
}

func NewMultistandardBalanceTokenListProvider(tokenManager *token.Manager) *MultistandardBalanceTokenListProvider {
	return &MultistandardBalanceTokenListProvider{tokenManager: tokenManager}
}

func (p *MultistandardBalanceTokenListProvider) GetTokenContractAddresses(chainID uint64) ([]common.Address, error) {
	tokens, err := p.tokenManager.GetTokens(chainID)
	if err != nil {
		return nil, err
	}

	addresses := make([]common.Address, 0)
	for _, token := range tokens {
		addresses = append(addresses, token.Address)
	}

	return addresses, nil
}

type MultistandardBalanceCollectiblesListProvider struct {
	storage        collectibles_ownership.OwnershipStorage
	contractTypeDB *collectibles.ContractTypeDB
}

func NewMultistandardBalanceCollectiblesListProvider(storage collectibles_ownership.OwnershipStorage, contractTypeDB *collectibles.ContractTypeDB) *MultistandardBalanceCollectiblesListProvider {
	return &MultistandardBalanceCollectiblesListProvider{storage: storage, contractTypeDB: contractTypeDB}
}

func (p *MultistandardBalanceCollectiblesListProvider) GetCollectiblesList(chainID uint64, account common.Address) (erc721 []multistandardbalance.CollectibleID, erc1155 []multistandardbalance.CollectibleID, err error) {
	const offset = 0
	const noLimit = -1
	ownedCollectibles, err := p.storage.GetOwnedCollectibles([]w_common.ChainID{w_common.ChainID(chainID)}, []common.Address{account}, offset, noLimit)
	if err != nil {
		return nil, nil, err
	}

	contracts := make([]thirdparty.ContractID, len(ownedCollectibles))
	for idx, collectible := range ownedCollectibles {
		contracts[idx] = collectible.ContractID
	}

	contractTypes, err := p.contractTypeDB.GetContractTypes(contracts)
	if err != nil {
		return nil, nil, err
	}

	for _, ownedCollectible := range ownedCollectibles {
		contractType := contractTypes[ownedCollectible.ContractID]
		if contractType == w_common.ContractTypeERC721 {
			erc721 = append(erc721, multistandardbalance.CollectibleID{
				ContractAddress: ownedCollectible.ContractID.Address,
				TokenID:         ownedCollectible.TokenID.Int,
			})
		} else if contractType == w_common.ContractTypeERC1155 {
			erc1155 = append(erc1155, multistandardbalance.CollectibleID{
				ContractAddress: ownedCollectible.ContractID.Address,
				TokenID:         ownedCollectible.TokenID.Int,
			})
		}
	}

	return erc721, erc1155, nil
}
