package puzzleauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
)

var retryStatusCodes = map[int]bool{
	http.StatusUnauthorized:    true, // 401
	http.StatusForbidden:       true, // 403
	http.StatusTooManyRequests: true, // 429
}

// Client is an HTTP client with automatic puzzle authentication
type Client struct {
	httpClient  *http.Client
	authService *Service
	maxRetries  int
}

// NewClient creates a new puzzle auth HTTP client
// If httpClient is nil, a default client with 60 second timeout will be created
func NewClient(origin string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}
	return &Client{
		httpClient:  httpClient,
		authService: NewService(origin, httpClient),
		maxRetries:  2, // Original attempt + 1 retry after auth
	}
}

// DoRequest executes an HTTP request with automatic puzzle authentication
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Try to get a token and add it to the request
		token := c.authService.GetToken()
		if token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}

		// Execute the request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		// Check if we need to retry with authentication
		if retryStatusCodes[resp.StatusCode] && attempt < c.maxRetries {
			// Read and close body before retrying
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			logutils.ZapLogger().Debug("Puzzle auth retry needed",
				zap.Int("statusCode", resp.StatusCode),
				zap.Int("attempt", attempt+1))

			// Invalidate current token
			c.authService.InvalidateToken()

			// Get a new token
			token, err := c.authService.EnsureToken(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get auth token: %w", err)
			}

			// Update the request header with new token
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			// Retry the request
			continue
		}

		// Success or non-retryable error
		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// DoGetRequest performs a GET request with puzzle authentication
func (c *Client) DoGetRequest(ctx context.Context, url string, params netUrl.Values) ([]byte, error) {
	if len(params) > 0 {
		url = url + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// GetAuthService returns the underlying auth service (useful for testing)
func (c *Client) GetAuthService() *Service {
	return c.authService
}
