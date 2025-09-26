package multistandardbalance

//go:generate go tool mockgen -package=mock_multistandardbalance -source=storage.go -destination=mock/storage.go

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Storage interface {
	UpdateNativeBalance(ctx context.Context, key BalancesKey, balance *big.Int, state State) (balanceChanged bool, oldState State, err error)
	GetNativeBalance(ctx context.Context, key BalancesKey) (balance *big.Int, state State, err error)
	UpdateERC20Balances(ctx context.Context, key BalancesKey, balances map[ContractAddress]*big.Int, state State) (balanceChanged bool, oldState State, err error)
	GetERC20Balances(ctx context.Context, key BalancesKey) (balances map[ContractAddress]*big.Int, state State, err error)
	UpdateERC721Balances(ctx context.Context, key BalancesKey, balances map[ContractAddress]*big.Int, state State) (balanceChanged bool, oldState State, err error)
	GetERC721Balances(ctx context.Context, key BalancesKey) (balances map[ContractAddress]*big.Int, state State, err error)
	UpdateERC1155Balances(ctx context.Context, key BalancesKey, balances map[HashableCollectibleID]*big.Int, state State) (balanceChanged bool, oldState State, err error)
	GetERC1155Balances(ctx context.Context, key BalancesKey) (balances map[HashableCollectibleID]*big.Int, state State, err error)

	ClearMissingAccounts(ctx context.Context, accounts []common.Address)
	ClearMissingChains(ctx context.Context, chains []uint64)
}
