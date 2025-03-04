package cryptocompare

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDs(t *testing.T) {
	stdClient := NewClient()
	require.Equal(t, baseID, stdClient.ID())

	clientWithParams := NewClientWithParams(Params{
		ID: "testID",
	})
	require.Equal(t, "testID", clientWithParams.ID())
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "base URL without trailing slash, path without leading slash",
			baseURL:  "https://example.com",
			path:     "api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL with trailing slash, path without leading slash",
			baseURL:  "https://example.com/",
			path:     "api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL without trailing slash, path with leading slash",
			baseURL:  "https://example.com",
			path:     "/api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL with trailing slash, path with leading slash",
			baseURL:  "https://example.com/",
			path:     "/api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "base URL with multiple trailing slashes",
			baseURL:  "https://example.com///",
			path:     "api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "path with multiple leading slashes",
			baseURL:  "https://example.com",
			path:     "///api/v1/endpoint",
			expected: "https://example.com/api/v1/endpoint",
		},
		{
			name:     "empty path",
			baseURL:  "https://example.com",
			path:     "",
			expected: "https://example.com/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the NewClientWithParams behavior which trims trailing slashes
			baseURL := strings.TrimSuffix(tc.baseURL, "/")
			client := &Client{baseURL: baseURL}

			result := client.buildURL(tc.path)
			require.Equal(t, tc.expected, result)
		})
	}
}
