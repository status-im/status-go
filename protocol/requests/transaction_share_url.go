package requests

import (
	"errors"
)

var (
	ErrInvalidTransactionType = errors.New("transaction-share-url: invalid transaction type")
)

type TransactionShareURL struct {
	TxType  int    `json:"txType"`
	Asset   string `json:"asset"`
	Amount  string `json:"amount"`
	Address string `json:"address"`
	ChainID int    `json:"chainId"`
	ToAsset string `json:"toAsset"`
}

func (r *TransactionShareURL) Validate() error {
	if r.TxType < 0 {
		return ErrInvalidTransactionType
	}

	return nil
}
