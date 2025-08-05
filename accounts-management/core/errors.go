package core

import "errors"

var (
	ErrLoggerIsMissing                        = errors.New("logger is missing")
	ErrAccountKeyStoreMissing                 = errors.New("account keystore is missing")
	ErrNoAccountSelected                      = errors.New("no account selected")
	ErrPersistenceIsMissing                   = errors.New("persistence is missing")
	ErrAccountDoesNotExist                    = errors.New("account doesn't exist")
	ErrAddressAndPasswordOrPrivateKeyRequired = errors.New("address and password or private key are required")
	ErrAccountIsNil                           = errors.New("account is nil")
	ErrKeypairIsNil                           = errors.New("keypair is nil")
	ErrCannotRemoveChatAccount                = errors.New("cannot remove chat account")
	ErrCannotRemoveDefaultWalletAccount       = errors.New("cannot remove default wallet account")
	ErrCannotRemoveProfileKeypair             = errors.New("cannot remove profile keypair")
)
