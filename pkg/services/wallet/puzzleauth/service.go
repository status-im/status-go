package puzzleauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
)

// Service manages puzzle authentication tokens
type Service struct {
	origin      string
	httpClient  *http.Client
	tokenCache  *TokenData
	mu          sync.RWMutex
	refreshLock sync.Mutex
}

// NewService creates a new puzzle auth service for the given origin
// If httpClient is nil, a default client with 30 second timeout will be created
func NewService(origin string, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Service{
		origin:     origin,
		httpClient: httpClient,
	}
}

// GetToken returns the current valid token (synchronously)
// Returns empty string if token is expired or doesn't exist
func (s *Service) GetToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tokenCache != nil && s.tokenCache.IsValid() {
		return s.tokenCache.Token
	}

	return ""
}

// InvalidateToken clears the cached token (tests and explicit full reset).
func (s *Service) InvalidateToken() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokenCache = nil
	logutils.ZapLogger().Debug("Puzzle auth token invalidated",
		zap.String("origin", s.origin))
}

// InvalidateTokenForRejectedRequest drops the cache only if it still holds the
// token that was used on a failed request. This avoids a thundering herd on 401/403/429
// clobbering a token another goroutine just refreshed.
// If no Bearer was sent (rejected is empty), this is a no-op; callers use EnsureToken only.
func (s *Service) InvalidateTokenForRejectedRequest(rejectedBearer string) {
	if rejectedBearer == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokenCache == nil || s.tokenCache.Token != rejectedBearer {
		return
	}
	s.tokenCache = nil
	logutils.ZapLogger().Debug("Puzzle auth token invalidated (rejected by server)",
		zap.String("origin", s.origin))
}

// EnsureToken returns a valid token, refreshing if necessary
func (s *Service) EnsureToken(ctx context.Context) (string, error) {
	if token := s.GetToken(); token != "" {
		return token, nil
	}

	// Need to refresh - use lock to prevent concurrent refreshes
	s.refreshLock.Lock()
	defer s.refreshLock.Unlock()

	// Double-check after acquiring lock (another goroutine might have refreshed)
	if token := s.GetToken(); token != "" {
		return token, nil
	}

	return s.refreshToken(ctx)
}

// refreshToken fetches a new token by solving a puzzle
func (s *Service) refreshToken(ctx context.Context) (string, error) {
	logutils.ZapLogger().Debug("Refreshing puzzle auth token",
		zap.String("origin", s.origin))

	startTime := time.Now()

	// Step 1: Get puzzle
	puzzle, err := s.getPuzzle(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get puzzle: %w", err)
	}

	// Step 2: Solve puzzle
	solution, err := Solve(ctx, puzzle)
	if err != nil {
		return "", fmt.Errorf("failed to solve puzzle: %w", err)
	}

	solveTime := time.Since(startTime)
	logutils.ZapLogger().Debug("Puzzle solved",
		zap.String("origin", s.origin),
		zap.Uint64("nonce", solution.Nonce),
		zap.Duration("solveTime", solveTime))

	// Step 3: Submit solution
	tokenResp, err := s.submitSolution(ctx, solution)
	if err != nil {
		return "", fmt.Errorf("failed to submit solution: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, tokenResp.ExpiresAt)
	if err != nil {
		logutils.ZapLogger().Warn("Failed to parse token expiry, using 1 hour default",
			zap.Error(err))
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	s.mu.Lock()
	s.tokenCache = &TokenData{
		Token:     tokenResp.Token,
		ExpiresAt: expiresAt,
	}
	s.mu.Unlock()

	logutils.ZapLogger().Debug("Puzzle auth token refreshed",
		zap.String("origin", s.origin),
		zap.Time("expiresAt", expiresAt),
		zap.Duration("totalTime", time.Since(startTime)))

	return tokenResp.Token, nil
}

// getPuzzle fetches a puzzle from the auth server
func (s *Service) getPuzzle(ctx context.Context) (*Puzzle, error) {
	url, err := netUrl.JoinPath(s.origin, "auth", "puzzle")
	if err != nil {
		return nil, fmt.Errorf("failed to construct puzzle URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("puzzle request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var puzzle Puzzle
	if err := json.NewDecoder(resp.Body).Decode(&puzzle); err != nil {
		return nil, err
	}

	return &puzzle, nil
}

// submitSolution submits a solved puzzle to get a JWT token
func (s *Service) submitSolution(ctx context.Context, solution *Solution) (*TokenResponse, error) {
	url, err := netUrl.JoinPath(s.origin, "auth", "solve")
	if err != nil {
		return nil, fmt.Errorf("failed to construct solve URL: %w", err)
	}

	jsonData, err := json.Marshal(solution)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("solution submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
