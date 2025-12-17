package keystore

import (
	"github.com/status-im/status-go/internal/accounts-management/errors"
)

const (
	ErrorCategoryKeystore errors.ErrorCategory = "keystore"
)

const (
	ErrCodeKeystoreFileMissing errors.ErrorCode = iota + 1
	ErrCodeIncorrectPasswordProvided
	ErrCodeInvalidAddress
)

var (
	ErrKeystoreFileMissing       = errors.NewError(ErrCodeKeystoreFileMissing, "keystore file is missing", getErrorCategory)
	ErrIncorrectPasswordProvided = errors.NewError(ErrCodeIncorrectPasswordProvided, "incorrect password provided", getErrorCategory)
	ErrInvalidAddress            = errors.NewError(ErrCodeInvalidAddress, "invalid address", getErrorCategory)
)

func getErrorCategory(code errors.ErrorCode) errors.ErrorCategory {
	switch code {
	case ErrCodeKeystoreFileMissing, ErrCodeIncorrectPasswordProvided, ErrCodeInvalidAddress:
		return ErrorCategoryKeystore
	default:
		return errors.ErrorCategoryUnknown
	}
}
