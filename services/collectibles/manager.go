package collectibles

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type Manager struct {
	rpcClient *rpc.Client
}

func NewManager(rpcClient *rpc.Client) *Manager {
	return &Manager{
		rpcClient: rpcClient,
	}
}

func (m *Manager) NewCollectiblesInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	backend, err := m.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return collectibles.NewCollectibles(common.HexToAddress(contractAddress), backend)
}

func (m *Manager) GetCollectiblesContractInstance(chainID uint64, contractAddress string) (*collectibles.Collectibles, error) {
	contractInst, err := m.NewCollectiblesInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst, nil
}

func (m *Manager) NewAssetsInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	backend, err := m.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	return assets.NewAssets(common.HexToAddress(contractAddress), backend)
}

func (m *Manager) GetAssetContractInstance(chainID uint64, contractAddress string) (*assets.Assets, error) {
	contractInst, err := m.NewAssetsInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	return contractInst, nil
}

func (m *Manager) GetCollectibleContractData(chainID uint64, contractAddress string) (*CollectibleContractData, error) {
	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	contract, err := m.GetCollectiblesContractInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	totalSupply, err := contract.TotalSupply(callOpts)
	if err != nil {
		return nil, err
	}
	transferable, err := contract.Transferable(callOpts)
	if err != nil {
		return nil, err
	}
	remoteBurnable, err := contract.RemoteBurnable(callOpts)
	if err != nil {
		return nil, err
	}

	return &CollectibleContractData{
		TotalSupply:    &bigint.BigInt{Int: totalSupply},
		Transferable:   transferable,
		RemoteBurnable: remoteBurnable,
		InfiniteSupply: GetInfiniteSupply().Cmp(totalSupply) == 0,
	}, nil
}

func (m *Manager) GetAssetContractData(chainID uint64, contractAddress string) (*AssetContractData, error) {
	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
	contract, err := m.GetAssetContractInstance(chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	totalSupply, err := contract.TotalSupply(callOpts)
	if err != nil {
		return nil, err
	}

	return &AssetContractData{
		TotalSupply:    &bigint.BigInt{Int: totalSupply},
		InfiniteSupply: GetInfiniteSupply().Cmp(totalSupply) == 0,
	}, nil
}
