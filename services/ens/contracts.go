package ens

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/erc20"
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

var sntByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"), // mainnet
	3: common.HexToAddress("0xc55cf4b03948d7ebc8b9e8bad92643703811d162"), // ropsten
}

type contractMaker struct {
	RPCClient *rpc.Client
}

func (c *contractMaker) newRegistry(chainID uint64) (*resolver.ENSRegistryWithFallback, error) {
	if _, ok := resolversByChainID[chainID]; !ok {
		return nil, errorNotAvailableOnChainID
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewENSRegistryWithFallback(
		resolversByChainID[chainID],
		backend,
	)
}

func (c *contractMaker) newPublicResolver(chainID uint64, resolverAddress *common.Address) (*resolver.PublicResolver, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewPublicResolver(*resolverAddress, backend)
}

func (c *contractMaker) newUsernameRegistrar(chainID uint64) (*registrar.UsernameRegistrar, error) {
	if _, ok := usernameRegistrarsByChainID[chainID]; !ok {
		return nil, errorNotAvailableOnChainID
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return registrar.NewUsernameRegistrar(
		usernameRegistrarsByChainID[chainID],
		backend,
	)
}

func (c *contractMaker) newSNT(chainID uint64) (*erc20.SNT, error) {
	if _, ok := sntByChainID[chainID]; !ok {
		return nil, errorNotAvailableOnChainID
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return erc20.NewSNT(sntByChainID[chainID], backend)
}
