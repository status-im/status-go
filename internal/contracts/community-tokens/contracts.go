package communitytokens

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/contracts/community-tokens/assets"
	"github.com/status-im/status-go/internal/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/internal/contracts/community-tokens/ownertoken"
	communityownertokenregistry "github.com/status-im/status-go/internal/contracts/community-tokens/registry"
	"github.com/status-im/status-go/internal/rpc"

	"github.com/status-im/status-go/internal/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/internal/contracts/community-tokens/mastertoken"
)

type CommunityTokensContractMaker struct {
	ethClientGetter rpc.EthClientGetter
}

func NewCommunityTokensContractMakerMaker(ethClientGetter rpc.EthClientGetter) *CommunityTokensContractMaker {
	return &CommunityTokensContractMaker{ethClientGetter: ethClientGetter}
}

func (c *CommunityTokensContractMaker) NewOwnerTokenInstance(chainID uint64, contractAddress common.Address,
) (*ownertoken.OwnerToken, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return ownertoken.NewOwnerToken(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewMasterTokenInstance(chainID uint64, contractAddress common.Address,
) (*mastertoken.MasterToken, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return mastertoken.NewMasterToken(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewCollectiblesInstance(chainID uint64, contractAddress common.Address,
) (*collectibles.Collectibles, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return collectibles.NewCollectibles(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewAssetsInstance(chainID uint64, contractAddress common.Address,
) (*assets.Assets, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return assets.NewAssets(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewCommunityTokenDeployerInstance(chainID uint64, contractAddress common.Address,
) (*communitytokendeployer.CommunityTokenDeployer, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return communitytokendeployer.NewCommunityTokenDeployer(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewCommunityOwnerTokenRegistryInstance(chainID uint64, contractAddress common.Address) (*communityownertokenregistry.CommunityOwnerTokenRegistry, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return communityownertokenregistry.NewCommunityOwnerTokenRegistry(contractAddress, backend)
}
