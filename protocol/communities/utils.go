package communities

import (
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
)

func CalculateRequestID(publicKey string, communityID types.HexBytes) types.HexBytes {
	idString := fmt.Sprintf("%s-%s", publicKey, communityID)
	return crypto.Keccak256([]byte(idString))
}

type TokenAddressesByChain = map[walletcommon.ChainID]map[gethcommon.Address]struct{}

func extractContractAddressesByChain(permissions []*CommunityTokenPermission) (erc20TokenAddresses TokenAddressesByChain, erc721TokenAddresses TokenAddressesByChain) {
	erc20TokenAddresses = TokenAddressesByChain{}
	erc721TokenAddresses = TokenAddressesByChain{}

	for _, tokenPermission := range permissions {
		for _, tokenRequirement := range tokenPermission.TokenCriteria {

			isERC721 := tokenRequirement.Type == protobuf.CommunityTokenType_ERC721
			isERC20 := tokenRequirement.Type == protobuf.CommunityTokenType_ERC20

			for chainID, contractAddress := range tokenRequirement.ContractAddresses {
				chainIDKey := walletcommon.ChainID(chainID)
				contractAddressKey := gethcommon.HexToAddress(contractAddress)

				if isERC721 {
					if erc721TokenAddresses[chainIDKey] == nil {
						erc721TokenAddresses[chainIDKey] = map[gethcommon.Address]struct{}{}
					}
					erc721TokenAddresses[chainIDKey][contractAddressKey] = struct{}{}
				}

				if isERC20 {
					if erc20TokenAddresses[chainIDKey] == nil {
						erc20TokenAddresses[chainIDKey] = map[gethcommon.Address]struct{}{}
					}
					erc20TokenAddresses[chainIDKey][contractAddressKey] = struct{}{}
				}
			}
		}
	}
	return
}
