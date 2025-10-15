package contracts

//go:generate go tool mockgen -source=contracts.go -destination=mock/contracts.go

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/directory"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/contracts/namewrapper"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/contracts/stickers"
	"github.com/status-im/status-go/rpc"
)

type ContractMakerIface interface {
	NewERC20(chainID uint64, contractAddr common.Address) (ierc20.IERC20Iface, error)
	NewERC20Caller(chainID uint64, contractAddr common.Address) (ierc20.IERC20CallerIface, error)
	// TODO extend with other contracts
}

type ContractMaker struct {
	ethClientGetter rpc.EthClientGetter
}

func NewContractMaker(client rpc.EthClientGetter) *ContractMaker {
	return &ContractMaker{ethClientGetter: client}
}

func (c *ContractMaker) NewRegistryWithAddress(chainID uint64, address common.Address) (*resolver.ENSRegistryWithFallback, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewENSRegistryWithFallback(
		address,
		backend,
	)
}

func (c *ContractMaker) NewRegistry(chainID uint64) (*resolver.ENSRegistryWithFallback, error) {
	contractAddr, err := resolver.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}
	return c.NewRegistryWithAddress(chainID, contractAddr)
}

func (c *ContractMaker) NewPublicResolver(chainID uint64, resolverAddress *common.Address) (*resolver.PublicResolver, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return resolver.NewPublicResolver(*resolverAddress, backend)
}

func (c *ContractMaker) NewUsernameRegistrar(chainID uint64, contractAddr common.Address) (*registrar.UsernameRegistrar, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return registrar.NewUsernameRegistrar(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewERC20(chainID uint64, contractAddr common.Address) (ierc20.IERC20Iface, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return ierc20.NewIERC20(
		contractAddr,
		backend,
	)
}
func (c *ContractMaker) NewERC20Caller(chainID uint64, contractAddr common.Address) (ierc20.IERC20CallerIface, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return ierc20.NewIERC20Caller(contractAddr, backend)
}

func (c *ContractMaker) NewSNT(chainID uint64) (*snt.SNT, error) {
	contractAddr, err := snt.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
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

	backend, err := c.ethClientGetter.EthClient(chainID)
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

	backend, err := c.ethClientGetter.EthClient(chainID)
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

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers.NewStickerPack(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewDirectory(chainID uint64) (*directory.Directory, error) {
	contractAddr, err := directory.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return directory.NewDirectory(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewNameWrapper(chainID uint64, address *common.Address) (*namewrapper.Namewrapper, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return namewrapper.NewNamewrapper(*address, backend)
}
