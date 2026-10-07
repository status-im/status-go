// Package requestgzip compresses request bodies for servers that accept it.
//
// HTTP has no way to ask a server up front whether it takes a compressed
// request body, so a server tells it in an Accept-Encoding header on its
// responses (RFC 7694). Transport sends plain bodies to a host until a
// response from it carries that header, and plain again once one does not.
// A host that turns a gzip body down all the same gets plain bodies from then on.
package requestgzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// MinSize is the smallest body worth compressing: below it the gzip header
// and the CPU outweigh what is saved.
const MinSize = 1 << 10

type Transport struct {
	base    http.RoundTripper
	accepts sync.Map // host -> struct{}
	refuses sync.Map // host -> struct{}
}

func New(base http.RoundTripper) *Transport {
	return &Transport{base: base}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host
	if !t.worthCompressing(req) {
		resp, err := t.base.RoundTrip(req)
		t.learn(host, resp)
		return resp, err
	}

	plain, err := readBody(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.base.RoundTrip(withBody(req, compress(plain), "gzip"))
	if err == nil && mayRefuseEncoding(resp.StatusCode) {
		refusal := resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		resp, err = t.base.RoundTrip(withBody(req, plain, ""))
		// The same answer to the plain body is about the request, not its encoding.
		if err == nil && resp.StatusCode != refusal {
			t.refuses.Store(host, struct{}{})
			t.accepts.Delete(host)
		}
	}
	t.learn(host, resp)
	return resp, err
}

// mayRefuseEncoding tells the statuses a server turns a gzip body down with:
// 415 by RFC 7694, 400 from those that take it for a malformed body. A server
// that could not read the body has not acted on it, so sending it again plain
// repeats nothing; what is sent here are reads and signed transactions, which
// a node takes once however often it gets them.
func mayRefuseEncoding(status int) bool {
	return status == http.StatusUnsupportedMediaType || status == http.StatusBadRequest
}

func (t *Transport) worthCompressing(req *http.Request) bool {
	if req.Body == nil || req.Body == http.NoBody || req.ContentLength < MinSize ||
		req.Header.Get("Content-Encoding") != "" {
		return false
	}
	_, ok := t.accepts.Load(req.URL.Host)
	return ok
}

// learn takes what the latest response from host says about gzip bodies.
func (t *Transport) learn(host string, resp *http.Response) {
	if resp == nil {
		return
	}
	if _, refused := t.refuses.Load(host); refused {
		return
	}
	if acceptsGzip(resp.Header.Values("Accept-Encoding")) {
		t.accepts.Store(host, struct{}{})
	} else {
		t.accepts.Delete(host)
	}
}

func acceptsGzip(values []string) bool {
	for _, value := range values {
		for _, coding := range strings.Split(value, ",") {
			name, params, _ := strings.Cut(coding, ";")
			if strings.EqualFold(strings.TrimSpace(name), "gzip") {
				return quality(params) > 0
			}
		}
	}
	return false
}

// quality is the q of a coding's parameters, 1 when it has none.
func quality(params string) float64 {
	for _, param := range strings.Split(params, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if ok && strings.EqualFold(key, "q") {
			q, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return 0
			}
			return q
		}
	}
	return 1
}

func readBody(req *http.Request) ([]byte, error) {
	if req.GetBody == nil {
		defer req.Body.Close()
		return io.ReadAll(req.Body)
	}
	_ = req.Body.Close()
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

var writers = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}

func compress(plain []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(plain) / 8)
	zw := writers.Get().(*gzip.Writer)
	defer writers.Put(zw)
	zw.Reset(&buf)
	_, _ = zw.Write(plain)
	_ = zw.Close()
	return buf.Bytes()
}

// withBody is req with body, which GetBody hands out again, under encoding.
func withBody(req *http.Request, body []byte, encoding string) *http.Request {
	out := req.Clone(req.Context())
	out.Body = io.NopCloser(bytes.NewReader(body))
	out.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	out.ContentLength = int64(len(body))
	if encoding == "" {
		out.Header.Del("Content-Encoding")
	} else {
		out.Header.Set("Content-Encoding", encoding)
	}
	return out
}
