package puzzleauth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenData_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		tokenData *TokenData
		expected  bool
	}{
		{
			name:      "nil token is invalid",
			tokenData: nil,
			expected:  false,
		},
		{
			name: "empty token is invalid",
			tokenData: &TokenData{
				Token:     "",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired token is invalid",
			tokenData: &TokenData{
				Token:     "valid-token",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "token expiring soon (within 5 min) is invalid",
			tokenData: &TokenData{
				Token:     "valid-token",
				ExpiresAt: time.Now().Add(3 * time.Minute),
			},
			expected: false,
		},
		{
			name: "valid token with sufficient time is valid",
			tokenData: &TokenData{
				Token:     "valid-token",
				ExpiresAt: time.Now().Add(10 * time.Minute),
			},
			expected: true,
		},
		{
			name: "valid token with 1 hour expiry is valid",
			tokenData: &TokenData{
				Token:     "valid-token",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokenData.IsValid()
			require.Equal(t, tt.expected, result)
		})
	}
}
