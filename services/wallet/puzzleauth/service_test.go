package puzzleauth

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestService_GetToken(t *testing.T) {
	service := NewService("https://test.nft.status.im", nil)

	token := service.GetToken()
	require.Empty(t, token)

	service.mu.Lock()
	service.tokenCache = &TokenData{
		Token:     "test-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	service.mu.Unlock()

	token = service.GetToken()
	require.Equal(t, "test-token", token)

	service.mu.Lock()
	service.tokenCache = &TokenData{
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	service.mu.Unlock()

	token = service.GetToken()
	require.Empty(t, token)
}

func TestService_InvalidateToken(t *testing.T) {
	service := NewService("https://test.nft.status.im", nil)

	service.mu.Lock()
	service.tokenCache = &TokenData{
		Token:     "test-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	service.mu.Unlock()

	require.NotEmpty(t, service.GetToken())

	service.InvalidateToken()

	require.Empty(t, service.GetToken())
	require.Nil(t, service.tokenCache)
}

func TestNewService(t *testing.T) {
	origin := "https://test.nft.status.im"
	service := NewService(origin, nil)

	require.NotNil(t, service)
	require.Equal(t, origin, service.origin)
	require.NotNil(t, service.httpClient)
	require.Nil(t, service.tokenCache)
}

func TestService_EnsureToken_FreshToken(t *testing.T) {
	var puzzleReq, solveReq int32
	server := newPuzzleAuthServer(t, withCounters(&puzzleReq, &solveReq))
	defer server.Close()

	service := NewService(server.URL, nil)
	ctx := context.Background()

	token, err := service.EnsureToken(ctx)
	require.NoError(t, err)
	require.Equal(t, "test-jwt-token", token)
	require.Equal(t, int32(1), atomic.LoadInt32(&puzzleReq))
	require.Equal(t, int32(1), atomic.LoadInt32(&solveReq))
	require.NotNil(t, service.tokenCache)
	require.Equal(t, "test-jwt-token", service.tokenCache.Token)
}

func TestService_EnsureToken_CachedToken(t *testing.T) {
	var puzzleReq int32
	server := newPuzzleAuthServer(t, withCounters(&puzzleReq, nil))
	defer server.Close()

	service := NewService(server.URL, nil)
	ctx := context.Background()

	service.mu.Lock()
	service.tokenCache = &TokenData{
		Token:     "cached-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	service.mu.Unlock()

	token, err := service.EnsureToken(ctx)
	require.NoError(t, err)
	require.Equal(t, "cached-token", token)
	require.Equal(t, int32(0), atomic.LoadInt32(&puzzleReq))
}

func TestService_EnsureToken_ConcurrentRequests(t *testing.T) {
	var puzzleReq, solveReq int32
	server := newPuzzleAuthServer(t, withCounters(&puzzleReq, &solveReq))
	defer server.Close()

	service := NewService(server.URL, nil)
	ctx := context.Background()

	const numRequests = 10
	var wg sync.WaitGroup
	wg.Add(numRequests)
	errors := make([]error, numRequests)
	tokens := make([]string, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			token, err := service.EnsureToken(ctx)
			tokens[idx] = token
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	for i := 0; i < numRequests; i++ {
		require.NoError(t, errors[i])
		require.Equal(t, "test-jwt-token", tokens[i])
	}

	require.Equal(t, int32(1), atomic.LoadInt32(&puzzleReq))
	require.Equal(t, int32(1), atomic.LoadInt32(&solveReq))
}

func TestService_RefreshToken_Errors(t *testing.T) {
	tests := []struct {
		name    string
		opts    []authServerOption
		wantErr string
	}{
		{
			name:    "puzzle error",
			opts:    []authServerOption{withPuzzleError(500)},
			wantErr: "failed to get puzzle",
		},
		{
			name:    "solve error",
			opts:    []authServerOption{withSolveError(400)},
			wantErr: "failed to submit solution",
		},
		{
			name:    "puzzle JSON error",
			opts:    []authServerOption{withPuzzleBody("invalid json")},
			wantErr: "failed to get puzzle",
		},
		{
			name:    "solve JSON error",
			opts:    []authServerOption{withSolveBody("invalid json")},
			wantErr: "failed to submit solution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newPuzzleAuthServer(t, tt.opts...)
			defer server.Close()

			service := NewService(server.URL, nil)
			ctx := context.Background()

			token, err := service.EnsureToken(ctx)
			require.Error(t, err)
			require.Empty(t, token)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestService_RefreshToken_InvalidExpiry(t *testing.T) {
	server := newPuzzleAuthServer(t, withInvalidExpiry())
	defer server.Close()

	service := NewService(server.URL, nil)
	ctx := context.Background()

	token, err := service.EnsureToken(ctx)
	require.NoError(t, err)
	require.Equal(t, "test-jwt-token", token)

	require.NotNil(t, service.tokenCache)
	require.Equal(t, "test-jwt-token", service.tokenCache.Token)
	timeDiff := service.tokenCache.ExpiresAt.Sub(time.Now())
	require.True(t, timeDiff > 55*time.Minute && timeDiff < 65*time.Minute,
		fmt.Sprintf("Expected expiry around 1 hour, got %v", timeDiff))
}
