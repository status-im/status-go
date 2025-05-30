package generator

import "errors"

var (
	ErrAccountNotFoundByID = errors.New("account not found")

	ErrAccountCannotDeriveChildKeys = errors.New("selected account cannot derive child keys")

	ErrAccountManagerNotSet = errors.New("account manager not set")
)
