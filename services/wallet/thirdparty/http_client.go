package thirdparty

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	netUrl "net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

const (
	defaultRequestTimeout  = 5 * time.Second
	defaultMaxRetries      = 5
	defaultIdleConnTimeout = 90 * time.Second
)

type BasicCreds struct {
	User     string
	Password string
}

// HTTPClient represents an HTTP client with configurable options
type HTTPClient struct {
	client     *http.Client
	maxRetries int
}

// Struct to hold request modifiers and data collectors
type requestModifiers struct {
	etag string
}

// RequestOption is a function that modifies an HTTP request
type RequestOption func(*http.Request, *requestModifiers)

func WithGzip() RequestOption {
	return func(req *http.Request, _ *requestModifiers) {
		req.Header.Set("Accept-Encoding", "gzip")
	}
}

func WithCredentials(creds *BasicCreds) RequestOption {
	return func(req *http.Request, _ *requestModifiers) {
		if creds != nil {
			req.SetBasicAuth(creds.User, creds.Password)
		}
	}
}

func WithHeader(key, value string) RequestOption {
	return func(req *http.Request, _ *requestModifiers) {
		req.Header.Set(key, value)
	}
}

func WithEtag(etag string) RequestOption {
	return func(req *http.Request, modifiers *requestModifiers) {
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
			modifiers.etag = etag
		}
	}
}

// Option defines a function type for configuring HTTPClient
type Option func(*HTTPClient)

// WithTimeout sets the overall request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *HTTPClient) {
		c.client.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries for failed requests
func WithMaxRetries(maxRetries int) Option {
	return func(c *HTTPClient) {
		c.maxRetries = maxRetries
	}
}

// WithDetailedTimeouts sets detailed timeouts for different connection phases
func WithDetailedTimeouts(dialTimeout, tlsHandshakeTimeout, responseHeaderTimeout, requestTimeout time.Duration) Option {
	return func(c *HTTPClient) {
		transport := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: dialTimeout,
			}).DialContext,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			IdleConnTimeout:       defaultIdleConnTimeout,
		}
		c.client.Transport = transport
		c.client.Timeout = requestTimeout
	}
}

// NewHTTPClient creates a new HTTPClient with the provided options
func NewHTTPClient(opts ...Option) *HTTPClient {
	client := &HTTPClient{
		client: &http.Client{
			Timeout: defaultRequestTimeout,
		},
		maxRetries: defaultMaxRetries,
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// doGetRequest performs a GET request with the given URL and parameters
// If creds is not nil, it will add basic auth to the request
// If etag is not empty, it will add an If-None-Match header to the request
// If the server responds with a 304 status code (`http.StatusNotModified`), it will return an empty body and the same etag
func (c *HTTPClient) doGetRequest(ctx context.Context, url string, params netUrl.Values, options ...RequestOption) (body []byte, newEtag string, err error) {
	startTime := time.Now()
	if len(params) > 0 {
		url = url + "?" + params.Encode()
	}

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logutils.ZapLogger().Debug("Failed to create GET request",
			zap.String("url", url),
			zap.Error(err))
		return
	}

	mods := &requestModifiers{}

	for _, option := range options {
		option(req, mods)
	}

	if mods.etag != "" {
		newEtag = mods.etag
	}

	var resp *http.Response
	maxRetries := c.maxRetries
	if maxRetries < 0 {
		maxRetries = defaultMaxRetries // Use default if not set
	}

	var retryCount int
	for i := 0; i < maxRetries; i++ {
		retryCount = i
		resp, err = c.client.Do(req)
		if err == nil || i == maxRetries-1 {
			break
		}
		logutils.ZapLogger().Debug("Retrying GET request after error",
			zap.String("url", url),
			zap.Int("retry", i+1),
			zap.Error(err))
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		logutils.ZapLogger().Debug("GET request failed after retries",
			zap.String("url", url),
			zap.Int("retries", retryCount),
			zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if mods.etag != "" && resp.StatusCode == http.StatusNotModified {
		return
	}

	newEtag = resp.Header.Get("Etag")

	body, err = c.readResponse(resp)
	if err != nil {
		logutils.ZapLogger().Debug("Failed to read GET response body",
			zap.String("url", url),
			zap.Error(err))
		return
	}

	duration := time.Since(startTime)
	logutils.ZapLogger().Debug("GET request completed",
		zap.String("url", url),
		zap.Int("status", resp.StatusCode),
		zap.Int("retries", retryCount),
		zap.Int("bodySize", len(body)),
		zap.Duration("duration", duration))

	return
}

func (c *HTTPClient) readResponse(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		var gzipErr error
		reader, gzipErr = gzip.NewReader(resp.Body)
		if gzipErr != nil {
			return nil, gzipErr
		}
		defer reader.Close()
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// DoGetRequest performs a GET request with the given URL and parameters
func (c *HTTPClient) DoGetRequest(ctx context.Context, url string, params netUrl.Values, options ...RequestOption) (body []byte, err error) {
	body, _, err = c.doGetRequest(ctx, url, params, options...)
	return
}

// DoGetRequestWithCredentials performs a GET request with the given URL and parameters
// If creds is not nil, it will add basic auth to the request
func (c *HTTPClient) DoGetRequestWithCredentials(ctx context.Context, url string, params netUrl.Values, creds *BasicCreds, options ...RequestOption) (body []byte, err error) {
	allOptions := []RequestOption{WithCredentials(creds)}
	allOptions = append(allOptions, options...)
	body, _, err = c.doGetRequest(ctx, url, params, allOptions...)
	return
}

// DoGetRequestWithEtag performs a GET request with the given URL and parameters
// If etag is not empty, it will add an If-None-Match header to the request
// If the server responds with a 304 status code (`http.StatusNotModified`), it will return an empty body and the same etag
func (c *HTTPClient) DoGetRequestWithEtag(ctx context.Context, url string, params netUrl.Values, etag string, options ...RequestOption) (body []byte, newEtag string, err error) {
	allOptions := []RequestOption{WithEtag(etag)}
	allOptions = append(allOptions, options...)
	return c.doGetRequest(ctx, url, params, allOptions...)
}

func (c *HTTPClient) DoPostRequest(ctx context.Context, url string, params map[string]interface{}, creds *BasicCreds) ([]byte, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	if creds != nil {
		req.SetBasicAuth(creds.User, creds.Password)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *HTTPClient) BuildURL(proxyURL, endpoint string) string {
	// Trim trailing slashes from proxy URL and leading slashes from endpoint
	baseURL := strings.TrimRight(proxyURL, "/")
	cleanEndpoint := strings.TrimLeft(endpoint, "/")
	return baseURL + "/" + cleanEndpoint
}
