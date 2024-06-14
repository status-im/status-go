package pathprocessor

import (
	"fmt"
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
)

func getSigner(chainID uint64, from types.Address, verifiedAccount *account.SelectedExtKey) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, verifiedAccount.AccountKey.PrivateKey)
	}
}

func makeKey(fromChain, toChain uint64, fromTokenSymbol, toTokenSymbol string) string {
	if fromTokenSymbol != "" || toTokenSymbol != "" {
		return fmt.Sprintf("%d-%d-%s-%s", fromChain, toChain, fromTokenSymbol, toTokenSymbol)
	}
	return fmt.Sprintf("%d-%d", fromChain, toChain)
}
