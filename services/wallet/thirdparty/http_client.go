package thirdparty

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	netUrl "net/url"
	"time"
)

const requestTimeout = 5 * time.Second
const maxNumOfRequestRetries = 5

type BasicCreds struct {
	User     string
	Password string
}

type HTTPClient struct {
	client     *http.Client
	maxRetries int
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: requestTimeout,
		},
		maxRetries: maxNumOfRequestRetries,
	}
}

// NewHTTPClientWithDetailedTimeouts creates a new HTTPClient with separate timeouts for
// connection establishment and data transfer
func NewHTTPClientWithDetailedTimeouts(
	dialTimeout time.Duration,
	tlsHandshakeTimeout time.Duration,
	responseHeaderTimeout time.Duration,
	requestTimeout time.Duration,
	maxRetries int,
) *HTTPClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: dialTimeout, // Timeout for establishing a connection
		}).DialContext,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,   // Timeout for TLS handshake
		ResponseHeaderTimeout: responseHeaderTimeout, // Timeout for receiving response headers
		IdleConnTimeout:       90 * time.Second,      // How long to keep idle connections
	}

	return &HTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   requestTimeout, // Overall request timeout
		},
		maxRetries: maxRetries,
	}
}

// doGetRequest performs a GET request with the given URL and parameters
// If creds is not nil, it will add basic auth to the request
// If etag is not empty, it will add an If-None-Match header to the request
// If the server responds with a 304 status code (`http.StatusNotModified`), it will return an empty body and the same etag
func (c *HTTPClient) doGetRequest(ctx context.Context, url string, params netUrl.Values, creds *BasicCreds, etag string) (body []byte, newEtag string, err error) {
	if len(params) > 0 {
		url = url + "?" + params.Encode()
	}

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	includeEtag := etag != ""
	if includeEtag {
		newEtag = etag
		req.Header.Add("If-None-Match", etag)
	}

	if creds != nil {
		req.SetBasicAuth(creds.User, creds.Password)
	}

	var resp *http.Response
	maxRetries := c.maxRetries
	if maxRetries <= 0 {
		maxRetries = maxNumOfRequestRetries // Use default if not set
	}

	for i := 0; i < maxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil || i == maxRetries-1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if includeEtag && resp.StatusCode == http.StatusNotModified {
		return
	}

	newEtag = resp.Header.Get("Etag")

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}

// DoGetRequest performs a GET request with the given URL and parameters
func (c *HTTPClient) DoGetRequest(ctx context.Context, url string, params netUrl.Values) (body []byte, err error) {
	body, _, err = c.doGetRequest(ctx, url, params, nil, "")
	return
}

// DoGetRequestWithCredentials performs a GET request with the given URL and parameters
// If creds is not nil, it will add basic auth to the request
func (c *HTTPClient) DoGetRequestWithCredentials(ctx context.Context, url string, params netUrl.Values, creds *BasicCreds) (body []byte, err error) {
	body, _, err = c.doGetRequest(ctx, url, params, creds, "")
	return
}

// DoGetRequestWithEtag performs a GET request with the given URL and parameters
// If etag is not empty, it will add an If-None-Match header to the request
// If the server responds with a 304 status code (`http.StatusNotModified`), it will return an empty body and the same etag
func (c *HTTPClient) DoGetRequestWithEtag(ctx context.Context, url string, params netUrl.Values, etag string) (body []byte, newEtag string, err error) {
	return c.doGetRequest(ctx, url, params, nil, etag)
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
