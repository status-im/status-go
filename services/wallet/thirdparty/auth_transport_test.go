package thirdparty

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/services/wallet/puzzleauth"
)

func TestAuthTransport_APIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no auth headers (API key is in URL)
		require.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	authParams := AuthParams{
		Type:   AuthTypeAPIKey,
		APIKey: security.NewSensitiveString("test-api-key"),
	}
	transport := NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, "test-provider")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthTransport_None_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	authParams := AuthParams{
		Type: AuthTypeNone,
	}
	transport := NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, "test-provider")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthTransport_Basic_SetsAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "testuser", username)
		require.Equal(t, "testpass", password)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	authParams := AuthParams{
		Type: AuthTypeBasic,
		Creds: &BasicCreds{
			User:     security.NewSensitiveString("testuser"),
			Password: security.NewSensitiveString("testpass"),
		},
	}
	transport := NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, "test-provider")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthTransport_Basic_RetryOn429(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success after retry"))
		}
	}))
	defer server.Close()

	authParams := AuthParams{
		Type: AuthTypeBasic,
		Creds: &BasicCreds{
			User:     security.NewSensitiveString("user"),
			Password: security.NewSensitiveString("pass"),
		},
	}
	transport := NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, "test-provider")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.GreaterOrEqual(t, attempts, 3, "Should have retried at least twice before succeeding")
	resp.Body.Close()
}

func newPuzzleAuthServerForTest(t *testing.T) *httptest.Server {
	t.Helper()
	puzzle := puzzleauth.Puzzle{
		Challenge:  "testchallenge",
		Salt:       hex.EncodeToString([]byte("testsalt12345678")),
		Difficulty: 1,
		HMAC:       "testhmac",
		ExpiresAt:  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		Argon2Params: puzzleauth.Argon2Params{
			MemoryKB: 64,
			Time:     1,
			Threads:  1,
			KeyLen:   32,
		},
	}
	tokenResp := puzzleauth.TokenResponse{
		Token:     "test-jwt-token",
		ExpiresAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/puzzle":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(puzzle)
		case "/auth/solve":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tokenResp)
		case "/resource":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestAuthTransport_POW_DelegatesToPuzzleAuth(t *testing.T) {
	server := newPuzzleAuthServerForTest(t)
	defer server.Close()

	puzzleClient := puzzleauth.NewClient(server.URL, nil)
	authParams := AuthParams{
		Type:             AuthTypePOW,
		PuzzleAuthClient: puzzleClient,
	}
	transport := NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, "test-provider")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/resource", nil)
	require.NoError(t, err)

	resp, err := transport.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthTransport_POW_NilPuzzleClient(t *testing.T) {
	authParams := AuthParams{
		Type:             AuthTypePOW,
		PuzzleAuthClient: nil,
	}
	transport := NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, "test-provider")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:1", nil)
	require.NoError(t, err)

	resp, err := transport.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.ErrorIs(t, err, ErrPuzzleAuthClientRequired)
}

func TestAuthTransport_Auth_Getter(t *testing.T) {
	apiKey := security.NewSensitiveString("my-key")
	creds := &BasicCreds{
		User:     security.NewSensitiveString("u"),
		Password: security.NewSensitiveString("p"),
	}

	tests := []struct {
		name       string
		authParams AuthParams
	}{
		{"APIKey", AuthParams{Type: AuthTypeAPIKey, APIKey: apiKey}},
		{"Basic", AuthParams{Type: AuthTypeBasic, Creds: creds}},
		{"None", AuthParams{Type: AuthTypeNone}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewAuthTransport(nil, tt.authParams, "provider")
			got := transport.Auth()
			require.Equal(t, tt.authParams.Type, got.Type)
			require.Equal(t, tt.authParams.APIKey.Reveal(), got.APIKey.Reveal())
			if tt.authParams.Creds != nil {
				require.NotNil(t, got.Creds)
				require.Equal(t, tt.authParams.Creds.User.Reveal(), got.Creds.User.Reveal())
				require.Equal(t, tt.authParams.Creds.Password.Reveal(), got.Creds.Password.Reveal())
			}
		})
	}
}

func TestAuthTransport_AuthTypeName(t *testing.T) {
	tests := []struct {
		name     string
		authType AuthType
		expected string
	}{
		{"APIKey", AuthTypeAPIKey, "APIKey"},
		{"Basic", AuthTypeBasic, "Basic"},
		{"POW", AuthTypePOW, "POW"},
		{"None", AuthTypeNone, "None"},
		{"Unknown", AuthType(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewAuthTransport(nil, AuthParams{Type: tt.authType}, "test")
			require.Equal(t, tt.expected, transport.authTypeName())
		})
	}
}

func TestNewAuthTransport_NilClient(t *testing.T) {
	transport := NewAuthTransport(nil, AuthParams{Type: AuthTypeNone}, "test")
	require.NotNil(t, transport)
	require.NotNil(t, transport.httpClient)
	require.Equal(t, time.Minute, transport.httpClient.Timeout)
}
