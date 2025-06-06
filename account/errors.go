package account

import (
	"errors"
)

var (
	ErrNoAccountSelected         = errors.New("no account has been selected, please login")
	ErrOnboardingNotStarted      = errors.New("onboarding must be started before choosing an account")
	ErrOnboardingAccountNotFound = errors.New("cannot find onboarding account with the given id")
	ErrAccountKeyStoreMissing    = errors.New("account key store is not set")
)
