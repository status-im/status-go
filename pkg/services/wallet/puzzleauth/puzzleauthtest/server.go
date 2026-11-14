// Package puzzleauthtest provides a fake puzzle auth server for tests of code
// that talks to a proxy through puzzleauth.
package puzzleauthtest

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/status-im/status-go/pkg/services/wallet/puzzleauth"
)

// Token is the JWT the server issues for a solved puzzle.
const Token = "test-jwt-token"

// NewServer starts a server that plays both the auth service and the proxy
// behind it: /auth/puzzle hands out a puzzle cheap enough to solve in a test,
// /auth/solve answers with [Token], and every other request goes to resource.
// The server is closed when the test ends.
func NewServer(t testing.TB, resource http.Handler) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/puzzle", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, puzzleauth.Puzzle{
			Challenge:  "testchallenge",
			Salt:       hex.EncodeToString([]byte("testsalt12345678")),
			Difficulty: 1,
			HMAC:       "testhmac",
			ExpiresAt:  time.Now().Add(time.Hour).Format(time.RFC3339),
			Argon2Params: puzzleauth.Argon2Params{
				MemoryKB: 64,
				Time:     1,
				Threads:  1,
				KeyLen:   32,
			},
		})
	})
	mux.HandleFunc("/auth/solve", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, puzzleauth.TokenResponse{
			Token:     Token,
			ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
		})
	})
	mux.Handle("/", resource)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
