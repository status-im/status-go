package pathprocessor

import (
	"fmt"
	"math/big"
	"strings"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func getSigner(chainID uint64, from types.Address, verifiedAccount *account.SelectedExtKey) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, verifiedAccount.AccountKey.PrivateKey)
	}
}

func makeKey(fromChain, toChain uint64, fromTokenSymbol, toTokenSymbol string, amount *big.Int) string {
	key := fmt.Sprintf("%d-%d", fromChain, toChain)
	if fromTokenSymbol != "" || toTokenSymbol != "" {
		key = fmt.Sprintf("%s-%s-%s", key, fromTokenSymbol, toTokenSymbol)
	}
	if amount != nil {
		key = fmt.Sprintf("%s-%s", key, amount.String())
	}
	return key
}

func getNameFromEnsUsername(ensUsername string) string {
	suffix := "." + walletCommon.StatusDomain
	if strings.HasSuffix(ensUsername, suffix) {
		return ensUsername[:len(ensUsername)-len(suffix)]
	}
	return ensUsername
}
