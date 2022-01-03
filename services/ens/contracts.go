package ens

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/registrar"
	"github.com/status-im/status-go/services/ens/resolver"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var resolversByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"), // mainnet
	3: common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"), // ropsten
}

var usernameRegistrarsByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0xDB5ac1a559b02E12F29fC0eC0e37Be8E046DEF49"), // mainnet
	3: common.HexToAddress("0xdaae165beb8c06e0b7613168138ebba774aff071"), // ropsten
}

type contractMaker struct {
	rpcClient *rpc.Client
}

func (c *contractMaker) newRegistry(chainID uint64) (*resolver.ENSRegistryWithFallback, error) {
	if _, ok := resolversByChainID[chainID]; !ok {
		return nil, errorNotAvailableOnChainID
	}

	backend, err := c.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewENSRegistryWithFallback(
		resolversByChainID[chainID],
		backend,
	)
}

func (c *contractMaker) newPublicResolver(chainID uint64, resolverAddress *common.Address) (*resolver.PublicResolver, error) {
	backend, err := c.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewPublicResolver(*resolverAddress, backend)
}

func (c *contractMaker) newUsernameRegistrar(chainID uint64) (*registrar.UsernameRegistrar, error) {
	if _, ok := usernameRegistrarsByChainID[chainID]; !ok {
		return nil, errorNotAvailableOnChainID
	}

	backend, err := c.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return registrar.NewUsernameRegistrar(
		usernameRegistrarsByChainID[chainID],
		backend,
	)
}
