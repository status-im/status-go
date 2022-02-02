package contracts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/contracts/stickers"
	"github.com/status-im/status-go/rpc"
)

type ContractMaker struct {
	RPCClient *rpc.Client
}

func (c *ContractMaker) NewRegistry(chainID uint64) (*resolver.ENSRegistryWithFallback, error) {
	contractAddr, err := resolver.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewENSRegistryWithFallback(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewPublicResolver(chainID uint64, resolverAddress *common.Address) (*resolver.PublicResolver, error) {
	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewPublicResolver(*resolverAddress, backend)
}

func (c *ContractMaker) NewUsernameRegistrar(chainID uint64) (*registrar.UsernameRegistrar, error) {
	contractAddr, err := registrar.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return registrar.NewUsernameRegistrar(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewSNT(chainID uint64) (*snt.SNT, error) {
	contractAddr, err := snt.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return snt.NewSNT(contractAddr, backend)
}

func (c *ContractMaker) NewStickerType(chainID uint64) (*stickers.StickerType, error) {
	contractAddr, err := stickers.StickerTypeContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers.NewStickerType(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewStickerMarket(chainID uint64) (*stickers.StickerMarket, error) {
	contractAddr, err := stickers.StickerMarketContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers.NewStickerMarket(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewStickerPack(chainID uint64) (*stickers.StickerPack, error) {
	contractAddr, err := stickers.StickerPackContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers.NewStickerPack(
		contractAddr,
		backend,
	)
}
