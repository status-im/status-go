package core

import "github.com/status-im/status-go/accounts-management/errors"

const (
	ErrorCategorySystem     errors.ErrorCategory = "system"
	ErrorCategoryAccount    errors.ErrorCategory = "account"
	ErrorCategoryKeystore   errors.ErrorCategory = "keystore"
	ErrorCategoryValidation errors.ErrorCategory = "validation"
	ErrorCategoryDatabase   errors.ErrorCategory = "database"
)

const (
	ErrCodeLoggerMissing errors.ErrorCode = iota + 1
	ErrCodeKeystoreMissing
	ErrCodePersistenceMissing
	ErrCodeAccountIsNil
	ErrCodeKeypairIsNil
	ErrCodeKeystoreFileMissing
	ErrCodeAccountMismatch
	ErrCodeAddressAndPasswordOrPrivateKeyRequired
	ErrCodeNoAccountSelected
	ErrCodeAccountDoesNotExist
	ErrCodeKeystoreDirectoryError
	ErrCodeKeypairDoesNotHaveWalletAccount
	ErrCodeUnsupportedWalletAccountPath
	ErrCodeKeypairAlreadyAdded
	ErrCodeAccountAlreadyAdded
	ErrCodeChatAccountNotFoundInDerivedAccounts
	ErrCodeKeypairMustHaveAtLeastOneWalletAccount
	ErrCodeCannotAddAccountsToKeypairImportedViaPrivateKey
	ErrCodeCannotAddDefaultWalletAccount
	ErrCodeCannotRemoveDefaultWalletAccount
	ErrCodeCannotAddDefaultChatAccount
	ErrCodeCannotRemoveDefaultChatAccount
	ErrCodeCannotMigrateProfileKeypair
	ErrCodeKeypairIsNotKeycard
	ErrCodeWrongPasswordProvided
	ErrCodeKeycardDoesNotHaveAnyAccounts
	ErrCodeKeycardDoesNotRelateToAnyKeypair
	ErrCodeCannotRemoveProfileKeypair
)

var (
	ErrLoggerIsMissing                                 = errors.NewError(ErrCodeLoggerMissing, "logger is missing", getErrorCategory)
	ErrKeystoreMissing                                 = errors.NewError(ErrCodeKeystoreMissing, "keystore is missing", getErrorCategory)
	ErrPersistenceMissing                              = errors.NewError(ErrCodePersistenceMissing, "persistence is missing", getErrorCategory)
	ErrAccountIsNil                                    = errors.NewError(ErrCodeAccountIsNil, "account is nil", getErrorCategory)
	ErrKeypairIsNil                                    = errors.NewError(ErrCodeKeypairIsNil, "keypair is nil", getErrorCategory)
	ErrKeystoreFileMissing                             = errors.NewError(ErrCodeKeystoreFileMissing, "keystore file is missing", getErrorCategory)
	ErrAccountMismatch                                 = errors.NewError(ErrCodeAccountMismatch, "account mismatch", getErrorCategory)
	ErrAddressAndPasswordOrPrivateKeyRequired          = errors.NewError(ErrCodeAddressAndPasswordOrPrivateKeyRequired, "address and password or private key are required", getErrorCategory)
	ErrNoAccountSelected                               = errors.NewError(ErrCodeNoAccountSelected, "no account selected", getErrorCategory)
	ErrAccountDoesNotExist                             = errors.NewError(ErrCodeAccountDoesNotExist, "account does not exist", getErrorCategory)
	ErrKeypairDoesNotHaveWalletAccount                 = errors.NewError(ErrCodeKeypairDoesNotHaveWalletAccount, "keypair does not have wallet account", getErrorCategory)
	ErrUnsupportedWalletAccountPath                    = errors.NewError(ErrCodeUnsupportedWalletAccountPath, "unsupported profile or seed imported key pair wallet account", getErrorCategory)
	ErrKeypairAlreadyAdded                             = errors.NewError(ErrCodeKeypairAlreadyAdded, "keypair already added", getErrorCategory)
	ErrAccountAlreadyAdded                             = errors.NewError(ErrCodeAccountAlreadyAdded, "account already added", getErrorCategory)
	ErrChatAccountNotFoundInDerivedAccounts            = errors.NewError(ErrCodeChatAccountNotFoundInDerivedAccounts, "chat account not found in derived accounts", getErrorCategory)
	ErrKeypairMustHaveAtLeastOneWalletAccount          = errors.NewError(ErrCodeKeypairMustHaveAtLeastOneWalletAccount, "keypair must have at least one wallet account", getErrorCategory)
	ErrCannotAddAccountsToKeypairImportedViaPrivateKey = errors.NewError(ErrCodeCannotAddAccountsToKeypairImportedViaPrivateKey, "cannot add accounts to keypair imported via private key", getErrorCategory)
	ErrCannotAddDefaultWalletAccount                   = errors.NewError(ErrCodeCannotAddDefaultWalletAccount, "cannot add default wallet account", getErrorCategory)
	ErrCannotRemoveDefaultWalletAccount                = errors.NewError(ErrCodeCannotRemoveDefaultWalletAccount, "cannot remove default wallet account", getErrorCategory)
	ErrCannotAddDefaultChatAccount                     = errors.NewError(ErrCodeCannotAddDefaultChatAccount, "cannot add default chat account", getErrorCategory)
	ErrCannotRemoveDefaultChatAccount                  = errors.NewError(ErrCodeCannotRemoveDefaultChatAccount, "cannot remove default chat account", getErrorCategory)
	ErrCannotMigrateProfileKeypair                     = errors.NewError(ErrCodeCannotMigrateProfileKeypair, "cannot migrate profile keypair", getErrorCategory)
	ErrKeypairIsNotKeycard                             = errors.NewError(ErrCodeKeypairIsNotKeycard, "keypair is not a keycard keypair", getErrorCategory)
	ErrKeycardDoesNotHaveAnyAccounts                   = errors.NewError(ErrCodeKeycardDoesNotHaveAnyAccounts, "keycard does not have any accounts", getErrorCategory)
	ErrCannotRemoveProfileKeypair                      = errors.NewError(ErrCodeCannotRemoveProfileKeypair, "cannot remove profile keypair", getErrorCategory)
)

func ErrKeystoreDirectoryError(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeKeystoreDirectoryError, "make keystore directory", err, getErrorCategory)
}

func ErrWrongPasswordProvided(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeWrongPasswordProvided, "wrong password provided", err, getErrorCategory)
}

func ErrKeycardDoesNotRelateToAnyKeypair(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeKeycardDoesNotRelateToAnyKeypair, "keycard does not relate to any keypair", err, getErrorCategory)
}

func getErrorCategory(code errors.ErrorCode) errors.ErrorCategory {
	switch code {
	case ErrCodeLoggerMissing, ErrCodeKeystoreMissing, ErrCodePersistenceMissing:
		return ErrorCategorySystem
	case ErrCodeAccountMismatch, ErrCodeNoAccountSelected, ErrCodeAccountDoesNotExist, ErrCodeAccountIsNil, ErrCodeKeypairIsNil:
		return ErrorCategoryAccount
	case ErrCodeKeystoreDirectoryError, ErrCodeKeystoreFileMissing:
		return ErrorCategoryKeystore
	case ErrCodeAddressAndPasswordOrPrivateKeyRequired, ErrCodeKeypairDoesNotHaveWalletAccount, ErrCodeUnsupportedWalletAccountPath,
		ErrCodeKeypairAlreadyAdded, ErrCodeAccountAlreadyAdded, ErrCodeChatAccountNotFoundInDerivedAccounts,
		ErrCodeKeypairMustHaveAtLeastOneWalletAccount, ErrCodeCannotAddAccountsToKeypairImportedViaPrivateKey,
		ErrCodeCannotMigrateProfileKeypair, ErrCodeKeypairIsNotKeycard, ErrCodeWrongPasswordProvided, ErrCodeKeycardDoesNotHaveAnyAccounts,
		ErrCodeKeycardDoesNotRelateToAnyKeypair:
		return ErrorCategoryValidation
	case ErrCodeCannotAddDefaultWalletAccount, ErrCodeCannotAddDefaultChatAccount, ErrCodeCannotRemoveDefaultWalletAccount,
		ErrCodeCannotRemoveDefaultChatAccount, ErrCodeCannotRemoveProfileKeypair:
		return ErrorCategoryDatabase
	default:
		return errors.ErrorCategoryUnknown
	}
}
