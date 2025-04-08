package leaderboard

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
)

// RequestHandler manages HTTP requests for market data
type RequestHandler struct {
	config ServiceConfig
	client *http.Client
}

// NewRequestHandler creates a new request handler with the given configuration
func NewRequestHandler(config ServiceConfig, client *http.Client) *RequestHandler {
	return &RequestHandler{
		config: config,
		client: client,
	}
}

// FetchData performs an HTTP request and processes the response
// Returns the response body, a success flag, and update statistics
func (h *RequestHandler) FetchData(ctx context.Context, endpoint string, etag *string, stats *Stats) ([]byte, bool) {
	// Create request
	req, err := h.createRequest(ctx, endpoint, etag)
	if err != nil {
		return nil, false
	}

	// Execute the request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	// Update statistics
	stats.TotalRequests++
	h.updateCacheStats(stats, resp)

	// Check for 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		stats.NotModifiedCount++
		return nil, false // No data update
	}

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	// Process response body
	body, updated := h.processResponseBody(stats, resp, etag)
	if !updated {
		return nil, false
	}

	return body, true
}

// createRequest creates a new HTTP request with proper headers
func (h *RequestHandler) createRequest(ctx context.Context, endpoint string, etag *string) (*http.Request, error) {
	url := h.config.ProxyURL + endpoint

	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication
	req.SetBasicAuth(h.config.Login, h.config.Password)

	// Add headers for features
	if h.config.AllowGzip {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	// Add ETag header if available
	if h.config.AllowETag && *etag != "" {
		req.Header.Set("If-None-Match", *etag)
	}

	return req, nil
}

// updateCacheStats updates cache-related statistics based on response headers
func (h *RequestHandler) updateCacheStats(stats *Stats, resp *http.Response) {
	if cacheStatus := resp.Header.Get("X-Proxy-Cache"); cacheStatus == "HIT" {
		stats.CacheHits++
	} else {
		stats.CacheMisses++
	}
}

// processResponseBody handles the response body, including gzip decompression
// Returns the body and a success flag
func (h *RequestHandler) processResponseBody(stats *Stats, resp *http.Response, etag *string) ([]byte, bool) {
	// Set up appropriate reader based on content encoding
	var reader io.ReadCloser = resp.Body
	if h.config.AllowGzip && resp.Header.Get("Content-Encoding") == "gzip" {
		stats.GzipResponseCount++
		var gzipErr error
		reader, gzipErr = gzip.NewReader(resp.Body)
		if gzipErr != nil {
			return nil, false
		}
		defer reader.Close()
	}

	// Store the ETag if enabled
	if h.config.AllowETag && resp.Header.Get("ETag") != "" {
		*etag = resp.Header.Get("ETag")
	}

	// Read the response body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, false
	}

	// Update total response size
	stats.TotalResponseSize += int64(len(body))

	return body, true
}
