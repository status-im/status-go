package puzzleauth

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testPuzzle() Puzzle {
	return Puzzle{
		Challenge:  "testchallenge",
		Salt:       hex.EncodeToString([]byte("testsalt12345678")),
		Difficulty: 1,
		HMAC:       "testhmac",
		ExpiresAt:  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		Argon2Params: Argon2Params{
			MemoryKB: 64,
			Time:     1,
			Threads:  1,
			KeyLen:   32,
		},
	}
}

func testTokenResponse(expiresAt string) TokenResponse {
	if expiresAt == "" {
		expiresAt = time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	}
	return TokenResponse{
		Token:     "test-jwt-token",
		ExpiresAt: expiresAt,
	}
}

type authServerConfig struct {
	puzzleStatus  int
	puzzleBody    string
	solveStatus   int
	solveBody     string
	resourceFunc  func(w http.ResponseWriter, r *http.Request)
	puzzleCounter *int32
	solveCounter  *int32
}

type authServerOption func(*authServerConfig)

func withPuzzleError(status int) authServerOption {
	return func(c *authServerConfig) {
		c.puzzleStatus = status
		c.puzzleBody = "error"
	}
}

func withSolveError(status int) authServerOption {
	return func(c *authServerConfig) {
		c.solveStatus = status
		c.solveBody = "error"
	}
}

func withPuzzleBody(body string) authServerOption {
	return func(c *authServerConfig) {
		c.puzzleBody = body
	}
}

func withSolveBody(body string) authServerOption {
	return func(c *authServerConfig) {
		c.solveBody = body
	}
}

func withInvalidExpiry() authServerOption {
	return func(c *authServerConfig) {
		c.solveBody = `{"token":"test-jwt-token","expires_at":"invalid-date"}`
	}
}

func withResourceHandler(fn func(w http.ResponseWriter, r *http.Request)) authServerOption {
	return func(c *authServerConfig) {
		c.resourceFunc = fn
	}
}

func withCounters(puzzleCounter, solveCounter *int32) authServerOption {
	return func(c *authServerConfig) {
		c.puzzleCounter = puzzleCounter
		c.solveCounter = solveCounter
	}
}

func newPuzzleAuthServer(t *testing.T, opts ...authServerOption) *httptest.Server {
	cfg := &authServerConfig{
		puzzleStatus: http.StatusOK,
		solveStatus:  http.StatusOK,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/puzzle":
			if cfg.puzzleCounter != nil {
				atomic.AddInt32(cfg.puzzleCounter, 1)
			}
			if cfg.puzzleStatus != http.StatusOK {
				w.WriteHeader(cfg.puzzleStatus)
				_, _ = w.Write([]byte(cfg.puzzleBody))
				return
			}
			if cfg.puzzleBody != "" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(cfg.puzzleBody))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(testPuzzle())

		case "/auth/solve":
			if cfg.solveCounter != nil {
				atomic.AddInt32(cfg.solveCounter, 1)
			}
			if cfg.solveStatus != http.StatusOK {
				w.WriteHeader(cfg.solveStatus)
				_, _ = w.Write([]byte(cfg.solveBody))
				return
			}
			if cfg.solveBody != "" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(cfg.solveBody))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(testTokenResponse(""))

		case "/resource":
			if cfg.resourceFunc != nil {
				cfg.resourceFunc(w, r)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			}

		default:
			http.NotFound(w, r)
		}
	}))
}
