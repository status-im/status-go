package httpbytes

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	unknownHost = "unknown"
	dirIn       = "in"
	dirOut      = "out"
)

var httpBytes = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "status_http_bytes",
		Help: "Outgoing HTTP request and response body bytes by host.",
	},
	[]string{"host", "dir"},
)

func init() {
	prometheus.MustRegister(httpBytes)
}

func Wrap(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if _, ok := base.(*countingTransport); ok {
		return base
	}
	return &countingTransport{base: base}
}

func WrapClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	wrapped := *client
	wrapped.Transport = Wrap(client.Transport)
	return &wrapped
}

func AddResponseBytes(host string, n int) {
	if n <= 0 {
		return
	}
	if host == "" {
		host = unknownHost
	}
	httpBytes.WithLabelValues(host, dirIn).Add(float64(n))
}

func HostFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return unknownHost
	}
	host := parsed.Host
	if host == "" {
		return unknownHost
	}
	return stripPort(host)
}

type countingTransport struct {
	base http.RoundTripper
}

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := HostLabel(req)
	if req.ContentLength > 0 {
		httpBytes.WithLabelValues(host, dirOut).Add(float64(req.ContentLength))
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp != nil && resp.Body != nil {
		resp.Body = &countingReadCloser{inner: resp.Body, host: host}
	}
	return resp, nil
}

func (t *countingTransport) CloseIdleConnections() {
	type idleCloser interface {
		CloseIdleConnections()
	}
	if closer, ok := t.base.(idleCloser); ok {
		closer.CloseIdleConnections()
	}
}

type countingReadCloser struct {
	inner io.ReadCloser
	host  string
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.inner.Read(p)
	if n > 0 {
		httpBytes.WithLabelValues(c.host, dirIn).Add(float64(n))
	}
	return n, err
}

func (c *countingReadCloser) Close() error {
	return c.inner.Close()
}

func HostLabel(req *http.Request) string {
	if req == nil {
		return unknownHost
	}
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	return stripPort(host)
}

func stripPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return unknownHost
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		if h == "" {
			return unknownHost
		}
		return h
	}
	return host
}
