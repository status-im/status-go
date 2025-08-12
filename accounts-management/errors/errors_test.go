package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountsError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AccountsError
		expected string
	}{
		{
			name:     "simple error without underlying error",
			err:      NewError(ErrCode1, "message 1", getErrorCategoryTest),
			expected: "[1] message 1",
		},
		{
			name:     "error with underlying error",
			err:      NewError(ErrCode2, "message 2", getErrorCategoryTest),
			expected: "[2] message 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAccountsError_Is(t *testing.T) {
	err1 := NewError(ErrCode1, "message 1", getErrorCategoryTest)
	err2 := NewError(ErrCode1, "different message", getErrorCategoryTest)
	err3 := NewError(ErrCode2, "message 2", getErrorCategoryTest)

	assert.True(t, err1.Is(err2), "errors with same code should match")
	assert.False(t, err1.Is(err3), "errors with different codes should not match")
}

func TestAccountsError_WithContext(t *testing.T) {
	err := NewError(ErrCode1, "invalid address", getErrorCategoryTest)
	err = err.WithContext("address", "0x123")
	err = err.WithContext("reason", "checksum failed")

	assert.Equal(t, "0x123", err.Context["address"])
	assert.Equal(t, "checksum failed", err.Context["reason"])
}

func TestErrorCode_Categories(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected ErrorCategory
	}{
		{ErrCode1, ErrorCategory1},
		{ErrCode2, ErrorCategory2},
		{ErrCode3, ErrorCategory3},
		{ErrCode4, ErrorCategory4},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			err := NewError(tt.code, "test error", getErrorCategoryTest)
			assert.Equal(t, tt.expected, err.Category)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("Err3", func(t *testing.T) {
		err := Err3("0x123", "0x456")

		assert.Equal(t, ErrCode3, err.Code)
		assert.Equal(t, ErrorCategory3, err.Category)
		assert.Equal(t, "0x123", err.Context["arg"])
		assert.Equal(t, "0x456", err.Context["arg2"])
	})
	t.Run("Err4", func(t *testing.T) {
		underlyingErr := errors.New("crypto error")
		err := Err4(underlyingErr)

		assert.Equal(t, ErrCode4, err.Code)
		assert.Equal(t, ErrorCategory4, err.Category)
		assert.Equal(t, underlyingErr, err.Err)
	})
}

func TestErrorComparison(t *testing.T) {
	err1 := NewError(ErrCode1, "message 1", getErrorCategoryTest)
	err2 := NewError(ErrCode1, "different message", getErrorCategoryTest)
	err3 := NewError(ErrCode2, "message 2", getErrorCategoryTest)

	assert.True(t, errors.Is(err1, err2))
	assert.False(t, errors.Is(err1, err3))

	assert.Equal(t, ErrCode1, err1.Code)
	assert.Equal(t, ErrCode1, err2.Code)
	assert.Equal(t, ErrCode2, err3.Code)
}
