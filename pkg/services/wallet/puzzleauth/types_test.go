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
			name: "token expiring within a minute is invalid",
			tokenData: &TokenData{
				Token:     "valid-token",
				ExpiresAt: time.Now().Add(30 * time.Second),
			},
			expected: false,
		},
		{
			name: "10-minute token is valid for 9 minutes",
			tokenData: &TokenData{
				Token:     "valid-token",
				IssuedAt:  time.Now().Add(-8 * time.Minute),
				ExpiresAt: time.Now().Add(2 * time.Minute),
			},
			expected: true,
		},
		{
			name: "10-minute token is invalid in its last minute",
			tokenData: &TokenData{
				Token:     "valid-token",
				IssuedAt:  time.Now().Add(-9*time.Minute - 10*time.Second),
				ExpiresAt: time.Now().Add(50 * time.Second),
			},
			expected: false,
		},
		{
			name: "short-lived token keeps nine tenths of its lifetime",
			tokenData: &TokenData{
				Token:     "valid-token",
				IssuedAt:  time.Now().Add(-1500 * time.Millisecond),
				ExpiresAt: time.Now().Add(500 * time.Millisecond),
			},
			expected: true,
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
