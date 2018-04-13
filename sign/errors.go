package sign

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/account"
)

// TODO (mandrigin): Change values of these errors when API change is made.
var (
	//ErrSignReqNotFound - error transaction hash not found
	ErrSignReqNotFound = errors.New("transaction hash not found")
	//ErrSignReqInProgress - error transaction is in progress
	ErrSignReqInProgress = errors.New("transaction is in progress")
	// TODO (mandrigin): to be moved to `transactions` package
	//ErrInvalidCompleteTxSender - error transaction with invalid sender
	ErrInvalidCompleteTxSender = errors.New("transaction can only be completed by the same account which created it")
	//ErrSignReqTimedOut - error transaction sending timed out
	ErrSignReqTimedOut = errors.New("transaction sending timed out")
	//ErrSignReqDiscarded - error transaction discarded
	ErrSignReqDiscarded = errors.New("transaction has been discarded")
)

// remove from queue on any error (except for transient ones) and propagate
// defined as map[string]bool because errors from ethclient returned wrapped as jsonError
var transientErrs = map[string]bool{
	keystore.ErrDecrypt.Error():          true, // wrong password
	ErrInvalidCompleteTxSender.Error():   true, // completing tx create from another account
	account.ErrNoAccountSelected.Error(): true, // account not selected
}

func isTransient(err error) bool {
	_, transient := transientErrs[err.Error()]
	return transient
}
