package generator

import "github.com/status-im/status-go/accounts-management/errors"

const (
	ErrorCategoryGenerator errors.ErrorCategory = "generator"
)

const (
	ErrCodeInvalidKeystoreExtendedKey errors.ErrorCode = iota + 1
	ErrCodeMnemonicCreationFailed
	ErrCodeAccountCreationFromMnemonicFailed
	ErrCodeAccountCreationFromPrivateKeyFailed
	ErrCodeAccountCreationFromKeyFailed
	ErrCodePathDecodingFailed
	ErrCodeZeroedExtendedKey
	ErrCodeExtendedKeyDerivationFailed
	ErrCodeChildAccountDerivationFailed
	ErrCodePathParsingFailed
	ErrCodeInvalidDerivationIndex
	ErrCodeUnexpectedToken
	ErrCodeUnexpectedEOF
)

var (
	ErrInvalidKeystoreExtendedKey = errors.NewError(ErrCodeInvalidKeystoreExtendedKey, "PrivateKey and ExtendedKey are different", getErrorCategory)
	ErrZeroedExtendedKey          = errors.NewError(ErrCodeZeroedExtendedKey, "can not derive child account from zeroed extended key", getErrorCategory)
	ErrPathParsingFailed          = errors.NewError(ErrCodePathParsingFailed, "error parsing derivation path", getErrorCategory)
	ErrInvalidDerivationIndex     = errors.NewError(ErrCodeInvalidDerivationIndex, "index must be lower than 2^31", getErrorCategory)
	ErrUnexpectedToken            = errors.NewError(ErrCodeUnexpectedToken, "unexpected token in derivation path", getErrorCategory)
	ErrUnexpectedEOF              = errors.NewError(ErrCodeUnexpectedEOF, "unexpected end of derivation path", getErrorCategory)
)

func ErrMnemonicCreationFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeMnemonicCreationFailed, "failed to create mnemonic seed", err, getErrorCategory)
}

func ErrAccountCreationFromMnemonicFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeAccountCreationFromMnemonicFailed, "can not create account from mnemonic", err, getErrorCategory)
}

func ErrAccountCreationFromPrivateKeyFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeAccountCreationFromPrivateKeyFailed, "can not create account from private key", err, getErrorCategory)
}

func ErrAccountCreationFromKeyFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeAccountCreationFromKeyFailed, "can not create account from key", err, getErrorCategory)
}

func ErrPathDecodingFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodePathDecodingFailed, "can not decode path", err, getErrorCategory)
}

func ErrExtendedKeyDerivationFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeExtendedKeyDerivationFailed, "can not derive child account from extended key", err, getErrorCategory)
}

func ErrChildAccountDerivationFailed(err error) *errors.AccountsError {
	return errors.WrapError(ErrCodeChildAccountDerivationFailed, "can not derive child account from path", err, getErrorCategory)
}

func getErrorCategory(code errors.ErrorCode) errors.ErrorCategory {
	switch code {
	case ErrCodeInvalidKeystoreExtendedKey,
		ErrCodeMnemonicCreationFailed,
		ErrCodeAccountCreationFromMnemonicFailed,
		ErrCodeAccountCreationFromPrivateKeyFailed,
		ErrCodeAccountCreationFromKeyFailed,
		ErrCodePathDecodingFailed,
		ErrCodeZeroedExtendedKey,
		ErrCodeExtendedKeyDerivationFailed,
		ErrCodeChildAccountDerivationFailed,
		ErrCodePathParsingFailed,
		ErrCodeInvalidDerivationIndex,
		ErrCodeUnexpectedToken,
		ErrCodeUnexpectedEOF:
		return ErrorCategoryGenerator
	default:
		return errors.ErrorCategoryUnknown
	}
}
