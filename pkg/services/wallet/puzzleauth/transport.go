package puzzleauth

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var retryStatusCodes = map[int]bool{
	http.StatusUnauthorized:    true, // 401
	http.StatusForbidden:       true, // 403
	http.StatusTooManyRequests: true, // 429
}

// shared auth service per origin: many eth RPC / Alchemy clients call NewTransport with the same
// host; a separate Service per client would N-fold puzzle solves and re-trigger 401 storms.
var sharedAuthServices sync.Map // string (origin) -> *Service

func sharedAuthServiceForOrigin(origin string) *Service {
	if s, ok := sharedAuthServices.Load(origin); ok {
		return s.(*Service)
	}
	authClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}
	svc := NewService(origin, authClient)
	if actual, loaded := sharedAuthServices.LoadOrStore(origin, svc); loaded {
		return actual.(*Service)
	}
	return svc
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
	return &Transport{
		base:        base,
		authService: sharedAuthServiceForOrigin(origin),
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

// readRequestBody materializes the body once for retries.
func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	return body, nil
}

// canStreamBody reports whether each attempt can send the request's own body:
// GetBody replays it for retries and the known length avoids chunked encoding.
func canStreamBody(req *http.Request) bool {
	return req.Body != nil && req.Body != http.NoBody && req.GetBody != nil && req.ContentLength > 0
}

// RoundTrip implements [http.RoundTripper].
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	stream := canStreamBody(req)
	var body []byte
	if !stream {
		var err error
		if body, err = readRequestBody(req); err != nil {
			return nil, err
		}
	}

	for attempt := 0; ; attempt++ {
		r := req.Clone(ctx)
		switch {
		case stream:
			if attempt > 0 {
				rc, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("puzzleauth: get request body: %w", err)
				}
				r.Body = rc
			}
			// Keep transport from using chunked encoding; some proxies reject that for JSON-RPC.
			r.TransferEncoding = nil
			r.Header.Del("Transfer-Encoding")
		case body != nil:
			b := body
			r.Body = io.NopCloser(bytes.NewReader(b))
			r.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(b)), nil
			}
			r.ContentLength = int64(len(b))
			r.TransferEncoding = nil
			r.Header.Del("Transfer-Encoding")
		default:
			r.ContentLength = 0
		}
		var usedBearer string
		if tok := t.authService.GetToken(); tok != "" {
			usedBearer = tok
			r.Header.Set("Authorization", "Bearer "+tok)
		}

		resp, err := t.base.RoundTrip(r)
		if err != nil {
			return nil, err
		}
		if !retryStatusCodes[resp.StatusCode] {
			return resp, nil
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if attempt >= t.maxRetries {
			return nil, fmt.Errorf("%w: status %d", ErrAuthRotating, resp.StatusCode)
		}
		t.authService.InvalidateTokenForRejectedRequest(usedBearer)
		if _, err = t.authService.EnsureToken(ctx); err != nil {
			return nil, fmt.Errorf("failed to get auth token: %w", err)
		}
	}
}
