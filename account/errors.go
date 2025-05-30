package account

import (
	"errors"
)

var (
	ErrAddressToAccountMappingFailure = errors.New("cannot retrieve a valid account for a given address")
	ErrAccountToKeyMappingFailure     = errors.New("cannot retrieve a valid key for a given account")
	ErrNoAccountSelected              = errors.New("no account has been selected, please login")
	ErrOnboardingNotStarted           = errors.New("onboarding must be started before choosing an account")
	ErrOnboardingAccountNotFound      = errors.New("cannot find onboarding account with the given id")
	ErrAccountKeyStoreMissing         = errors.New("account key store is not set")
)
