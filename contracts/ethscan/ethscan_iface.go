package ethscan

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type BalanceScannerIface interface {
	EtherBalances(opts *bind.CallOpts, addresses []common.Address) ([]BalanceScannerResult, error)
	TokenBalances(opts *bind.CallOpts, addresses []common.Address, tokenAddress common.Address) ([]BalanceScannerResult, error)
	TokensBalance(opts *bind.CallOpts, owner common.Address, contracts []common.Address) ([]BalanceScannerResult, error)
}

// Verify that BalanceScanner implements BalanceScannerIface. If contract changes, this will fail to compile.
var _ BalanceScannerIface = (*BalanceScanner)(nil)
