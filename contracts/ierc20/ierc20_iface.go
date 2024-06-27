package ierc20

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type IERC20Iface interface {
	BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error)
	Name(opts *bind.CallOpts) (string, error)
	Symbol(opts *bind.CallOpts) (string, error)
	Decimals(opts *bind.CallOpts) (uint8, error)
	Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error)
}

// Verify that IERC20 implements IERC20Iface. If contract changes, this will fail to compile, update interface to match.
var _ IERC20Iface = (*IERC20)(nil)

type IERC20CallerIface interface {
	BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error)
	Name(opts *bind.CallOpts) (string, error)
	Symbol(opts *bind.CallOpts) (string, error)
	Decimals(opts *bind.CallOpts) (uint8, error)
}

var _ IERC20CallerIface = (*IERC20Caller)(nil)
