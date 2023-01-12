package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
)

func GetSigner(chainID uint64, accountsManager *account.GethManager, keyStoreDir string, from types.Address, password string) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		selectedAccount, err := accountsManager.VerifyAccountPassword(keyStoreDir, from.Hex(), password)
		if err != nil {
			return nil, err
		}
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, selectedAccount.PrivateKey)
	}
}
