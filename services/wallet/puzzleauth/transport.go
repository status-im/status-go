package puzzleauth

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
)

var retryStatusCodes = map[int]bool{
	http.StatusUnauthorized:    true, // 401
	http.StatusForbidden:       true, // 403
	http.StatusTooManyRequests: true, // 429
}

// Transport is an [http.RoundTripper] that adds puzzle JWT (Bearer) auth and retries on 401/403/429.
type Transport struct {
	base        http.RoundTripper
	authService *Service
	maxRetries  int
}

// NewTransport returns a [Transport] for the given auth server origin. Actual requests are sent via base;
// if base is nil, [http.DefaultTransport] is used. Fetching /auth/puzzle and /auth/solve uses a
// separate client (default round tripper) so the puzzle layer never calls itself.
func NewTransport(origin string, base http.RoundTripper) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	authClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}
	return &Transport{
		base:        base,
		authService: NewService(origin, authClient),
		maxRetries:  2, // original attempt + 2 retries after auth
	}
}

// NewHTTPClient returns an [*http.Client] with [Transport] set to [NewTransport](origin, nil) and a 60s timeout.
func NewHTTPClient(origin string) *http.Client {
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: NewTransport(origin, nil),
	}
}

// RoundTrip implements [http.RoundTripper].
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	if req.Body != nil && req.GetBody == nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		clonedReq := req.Clone(ctx)
		if req.GetBody != nil && clonedReq.Body == nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to recreate request body: %w", err)
			}
			clonedReq.Body = body
		}

		token := t.authService.GetToken()
		if token != "" {
			clonedReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}

		resp, err := t.base.RoundTrip(clonedReq)
		if err != nil {
			return nil, err
		}

		if retryStatusCodes[resp.StatusCode] && attempt < t.maxRetries {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			logutils.ZapLogger().Debug("Puzzle auth retry needed",
				zap.Int("statusCode", resp.StatusCode),
				zap.Int("attempt", attempt+1))

			t.authService.InvalidateToken()

			token, err := t.authService.EnsureToken(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get auth token: %w", err)
			}

			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("puzzle auth: unexpected state after %d attempts", t.maxRetries+1)
}
