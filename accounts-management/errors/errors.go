package errors

import (
	"errors"
	"fmt"
)

type ErrorCode int

type ErrorCategory string

const (
	ErrorCategoryUnknown ErrorCategory = "unknown"
)

type AccountsError struct {
	Code     ErrorCode
	Category ErrorCategory
	Message  string
	Err      error
	Context  map[string]interface{}
}

// NewError creates a new AccountsError with the given code and message
func NewError(code ErrorCode, message string, getErrorCategory func(code ErrorCode) ErrorCategory) *AccountsError {
	category := ErrorCategoryUnknown
	if getErrorCategory != nil {
		category = getErrorCategory(code)
	}
	return &AccountsError{
		Code:     code,
		Category: category,
		Message:  message,
	}
}

// WrapError wraps an existing error with additional context
func WrapError(code ErrorCode, message string, err error, getErrorCategory func(code ErrorCode) ErrorCategory) *AccountsError {
	accountsErr := NewError(code, message, getErrorCategory)
	accountsErr.Err = err
	return accountsErr
}

// WithContext adds additional context to the error
func (e *AccountsError) WithContext(key string, value interface{}) *AccountsError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Error implements the error interface
func (e *AccountsError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category, e.Message, e.Err)
	}
	if e.Context != nil {
		message := ""
		for k, v := range e.Context {
			message += fmt.Sprintf(" %s: %v", k, v)
		}
		return fmt.Sprintf("[%s] %s - %s", e.Category, e.Message, message)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

// Is checks if the error matches a specific error code
func (e *AccountsError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check if target is an AccountsError with the same code
	var targetAccountsErr *AccountsError
	if errors.As(target, &targetAccountsErr) {
		return e.Code == targetAccountsErr.Code && e.Category == targetAccountsErr.Category
	}

	return false
}
