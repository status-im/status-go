package wallet

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	collectibles "github.com/status-im/status-go/pkg/services/wallet/collectibles"
	collectibles_ownership "github.com/status-im/status-go/pkg/services/wallet/collectibles/ownership"
	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/token"
)

type MultistandardBalanceTokenListProvider struct {
	tokenManager *token.Manager
}

func NewMultistandardBalanceTokenListProvider(tokenManager *token.Manager) *MultistandardBalanceTokenListProvider {
	return &MultistandardBalanceTokenListProvider{tokenManager: tokenManager}
}

func (p *MultistandardBalanceTokenListProvider) GetTokenContractAddresses(chainID uint64) ([]common.Address, error) {
	tokens, err := p.tokenManager.GetTokensByChain(chainID)
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

// MultistandardBalanceHeldTokensProvider gives the custom tokens of a chain and
// the tokens of the pending ERC20 transfers of an account.
type MultistandardBalanceHeldTokensProvider struct {
	tokenManager     *token.Manager
	pendingTxTracker *pendingtxtracker.PendingTxTracker
}

func NewMultistandardBalanceHeldTokensProvider(tokenManager *token.Manager, pendingTxTracker *pendingtxtracker.PendingTxTracker) *MultistandardBalanceHeldTokensProvider {
	return &MultistandardBalanceHeldTokensProvider{tokenManager: tokenManager, pendingTxTracker: pendingTxTracker}
}

func (p *MultistandardBalanceHeldTokensProvider) GetHeldTokens(chainID uint64, account common.Address) ([]common.Address, error) {
	customs, err := p.tokenManager.GetCustoms(false)
	if err != nil {
		return nil, err
	}
	var held []common.Address
	for _, custom := range customs {
		if custom.ChainID == chainID {
			held = append(held, custom.Address)
		}
	}
	if p.pendingTxTracker == nil {
		return held, nil
	}
	pending, err := p.pendingTxTracker.GetAllPending()
	if err != nil {
		return held, err
	}
	return append(held, pendingERC20Transfers(pending, chainID, account)...), nil
}

// erc20TransferSelectors are the selectors of transfer and transferFrom.
var erc20TransferSelectors = []string{"0xa9059cbb", "0x23b872dd"}

// pendingERC20Transfers are the tokens account sends on chainID in pending
// transactions.
func pendingERC20Transfers(pending []*pendingtxtracker.PendingTransaction, chainID uint64, account common.Address) []common.Address {
	var tokens []common.Address
	for _, tx := range pending {
		if uint64(tx.ChainID) != chainID || tx.From != account {
			continue
		}
		data := strings.ToLower(tx.Data)
		for _, selector := range erc20TransferSelectors {
			if strings.HasPrefix(data, selector) {
				tokens = append(tokens, tx.To)
				break
			}
		}
	}
	return tokens
}

// TransferDetectionKnownTokensProvider gives the tokens whose transfers are
// kept: the token lists and custom tokens.
type TransferDetectionKnownTokensProvider struct {
	tokenManager *token.Manager
}

func NewTransferDetectionKnownTokensProvider(tokenManager *token.Manager) *TransferDetectionKnownTokensProvider {
	return &TransferDetectionKnownTokensProvider{tokenManager: tokenManager}
}

func (p *TransferDetectionKnownTokensProvider) GetTokenContractAddresses(chainID uint64) ([]common.Address, error) {
	listed, err := p.tokenManager.GetTokensByChain(chainID)
	if err != nil {
		return nil, err
	}
	customs, err := p.tokenManager.GetCustoms(false)
	if err != nil {
		return nil, err
	}
	known := make([]common.Address, 0, len(listed)+len(customs))
	for _, t := range listed {
		known = append(known, t.Address)
	}
	for _, custom := range customs {
		if custom.ChainID == chainID {
			known = append(known, custom.Address)
		}
	}
	return known, nil
}
