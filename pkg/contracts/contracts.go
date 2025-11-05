package contracts

//go:generate go tool mockgen -source=contracts.go -destination=mock/contracts.go

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/contracts/stickers"

	directory2 "github.com/status-im/status-go/pkg/contracts/directory"
	ierc21 "github.com/status-im/status-go/pkg/contracts/ierc20"
	"github.com/status-im/status-go/pkg/contracts/namewrapper"
	resolver2 "github.com/status-im/status-go/pkg/contracts/resolver"
	snt2 "github.com/status-im/status-go/pkg/contracts/snt"
	stickers2 "github.com/status-im/status-go/pkg/contracts/stickers"
	"github.com/status-im/status-go/rpc"
)

type ContractMakerIface interface {
	NewERC20(chainID uint64, contractAddr common.Address) (ierc21.IERC20Iface, error)
	NewERC20Caller(chainID uint64, contractAddr common.Address) (ierc21.IERC20CallerIface, error)
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
	contractAddr, err := resolver2.ContractAddress(chainID)
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

func (c *ContractMaker) NewERC20(chainID uint64, contractAddr common.Address) (ierc21.IERC20Iface, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return ierc20.NewIERC20(
		contractAddr,
		backend,
	)
}
func (c *ContractMaker) NewERC20Caller(chainID uint64, contractAddr common.Address) (ierc21.IERC20CallerIface, error) {
	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return ierc20.NewIERC20Caller(contractAddr, backend)
}

func (c *ContractMaker) NewSNT(chainID uint64) (*snt.SNT, error) {
	contractAddr, err := snt2.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return snt.NewSNT(contractAddr, backend)
}

func (c *ContractMaker) NewStickerType(chainID uint64) (*stickers2.StickerType, error) {
	contractAddr, err := stickers.StickerTypeContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers2.NewStickerType(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewStickerMarket(chainID uint64) (*stickers2.StickerMarket, error) {
	contractAddr, err := stickers.StickerMarketContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers2.NewStickerMarket(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewStickerPack(chainID uint64) (*stickers2.StickerPack, error) {
	contractAddr, err := stickers.StickerPackContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return stickers2.NewStickerPack(
		contractAddr,
		backend,
	)
}

func (c *ContractMaker) NewDirectory(chainID uint64) (*directory2.Directory, error) {
	contractAddr, err := directory2.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	backend, err := c.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return directory2.NewDirectory(
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
