package communitytokens

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/contracts/community-tokens/mastertoken"
	"github.com/status-im/status-go/contracts/community-tokens/ownertoken"
	communityownertokenregistry "github.com/status-im/status-go/contracts/community-tokens/registry"
	"github.com/status-im/status-go/rpc"
)

type CommunityTokensContractMaker struct {
	RPCClient rpc.ClientInterface
}

func NewCommunityTokensContractMakerMaker(client rpc.ClientInterface) (*CommunityTokensContractMaker, error) {
	if client == nil {
		return nil, errors.New("rpc client is required")
	}
	return &CommunityTokensContractMaker{RPCClient: client}, nil
}

func (c *CommunityTokensContractMaker) NewOwnerTokenInstance(chainID uint64, contractAddress common.Address,
) (*ownertoken.OwnerToken, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return ownertoken.NewOwnerToken(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewMasterTokenInstance(chainID uint64, contractAddress common.Address,
) (*mastertoken.MasterToken, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return mastertoken.NewMasterToken(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewCollectiblesInstance(chainID uint64, contractAddress common.Address,
) (*collectibles.Collectibles, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return collectibles.NewCollectibles(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewAssetsInstance(chainID uint64, contractAddress common.Address,
) (*assets.Assets, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return assets.NewAssets(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewCommunityTokenDeployerInstance(chainID uint64, contractAddress common.Address,
) (*communitytokendeployer.CommunityTokenDeployer, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return communitytokendeployer.NewCommunityTokenDeployer(contractAddress, backend)
}

func (c *CommunityTokensContractMaker) NewCommunityOwnerTokenRegistryInstance(chainID uint64, contractAddress common.Address) (*communityownertokenregistry.CommunityOwnerTokenRegistry, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return communityownertokenregistry.NewCommunityOwnerTokenRegistry(contractAddress, backend)
}
