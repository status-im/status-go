package requestgzip

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// received is what a server got: the encoding, the size on the wire and the
// body once decoded.
type received struct {
	encoding string
	wireSize int
	body     string
}

type server struct {
	*httptest.Server
	mu        sync.Mutex
	got       []received
	advertise string
	reject    bool
	// rejectStatus answers a refused gzip body, 415 when zero; status answers
	// everything else, 200 when zero.
	rejectStatus int
	status       int
	hits         int
}

func newServer(t *testing.T, advertise string) *server {
	s := &server{advertise: advertise}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wire, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		body := wire
		encoding := r.Header.Get("Content-Encoding")
		s.mu.Lock()
		defer s.mu.Unlock()
		s.hits++
		if s.advertise != "" {
			w.Header().Set("Accept-Encoding", s.advertise)
		}
		if encoding == "gzip" {
			if s.reject {
				w.WriteHeader(cmp.Or(s.rejectStatus, http.StatusUnsupportedMediaType))
				return
			}
			zr, err := gzip.NewReader(bytes.NewReader(wire))
			require.NoError(t, err)
			body, err = io.ReadAll(zr)
			require.NoError(t, err)
		}
		s.got = append(s.got, received{encoding: encoding, wireSize: len(wire), body: string(body)})
		w.WriteHeader(cmp.Or(s.status, http.StatusOK))
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x"}`))
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *server) last() received {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.got[len(s.got)-1]
}

func (s *server) setAdvertise(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advertise = v
}

func (s *server) setReject(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reject = v
}

var largeBody = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x` + strings.Repeat("00000000000000000000000070a08231", 200) + `"},"latest"]}`

func post(t *testing.T, client *http.Client, url, body string) {
	require.Equal(t, http.StatusOK, postStatus(t, client, url, body))
}

func postStatus(t *testing.T, client *http.Client, url, body string) int {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

func (s *server) set(change func(*server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(s)
}

func (s *server) hitCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

func TestTransport_SendsPlainUntilTheServerSaysItAcceptsGzip(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}

	post(t, client, s.URL, largeBody)
	require.Equal(t, "", s.last().encoding, "nothing is known about the server yet")

	post(t, client, s.URL, largeBody)
	got := s.last()
	require.Equal(t, "gzip", got.encoding)
	require.Equal(t, largeBody, got.body)
	require.Less(t, got.wireSize, len(largeBody)/10)
}

func TestTransport_NeverCompressesSmallBodies(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	small := `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`

	post(t, client, s.URL, small)
	post(t, client, s.URL, small)

	require.Equal(t, "", s.last().encoding)
	require.Equal(t, small, s.last().body)
}

func TestTransport_NeverCompressesForAServerThatDoesNotAccept(t *testing.T) {
	for _, advertise := range []string{"", "identity", "br", "gzip;q=0", "gzip; q=0.000"} {
		s := newServer(t, advertise)
		client := &http.Client{Transport: New(http.DefaultTransport)}

		post(t, client, s.URL, largeBody)
		post(t, client, s.URL, largeBody)

		require.Equal(t, "", s.last().encoding, advertise)
	}
}

func TestTransport_ReadsAcceptEncodingLists(t *testing.T) {
	s := newServer(t, "br, GZIP;q=0.8")
	client := &http.Client{Transport: New(http.DefaultTransport)}

	post(t, client, s.URL, largeBody)
	post(t, client, s.URL, largeBody)

	require.Equal(t, "gzip", s.last().encoding)
}

func TestTransport_LearnsEachHostOnItsOwn(t *testing.T) {
	accepting := newServer(t, "gzip")
	plain := newServer(t, "")
	client := &http.Client{Transport: New(http.DefaultTransport)}

	post(t, client, accepting.URL, largeBody)
	post(t, client, plain.URL, largeBody)
	post(t, client, accepting.URL, largeBody)
	post(t, client, plain.URL, largeBody)

	require.Equal(t, "gzip", accepting.last().encoding)
	require.Equal(t, "", plain.last().encoding)
}

func TestTransport_StopsCompressingWhenTheServerStopsAccepting(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	post(t, client, s.URL, largeBody)

	s.setAdvertise("")
	post(t, client, s.URL, largeBody)
	require.Equal(t, "gzip", s.last().encoding, "the last answer still said gzip")
	post(t, client, s.URL, largeBody)
	require.Equal(t, "", s.last().encoding)
}

func TestTransport_ResendsPlainWhenGzipIsRefused(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	post(t, client, s.URL, largeBody)

	s.setReject(true)
	s.setAdvertise("")
	post(t, client, s.URL, largeBody)
	require.Equal(t, "", s.last().encoding)
	require.Equal(t, largeBody, s.last().body)

	post(t, client, s.URL, largeBody)
	require.Equal(t, "", s.last().encoding, "a refusal is remembered")
}

func TestTransport_RemembersARefusalFromAServerThatStillAdvertisesGzip(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	post(t, client, s.URL, largeBody)

	s.setReject(true)
	post(t, client, s.URL, largeBody)
	require.Equal(t, "", s.last().encoding)

	before := s.hitCount()
	post(t, client, s.URL, largeBody)
	require.Equal(t, "", s.last().encoding)
	require.Equal(t, before+1, s.hitCount(), "the body is sent once, plain")
}

func TestTransport_ResendsPlainWhenGzipIsABadRequest(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	post(t, client, s.URL, largeBody)

	s.set(func(s *server) { s.reject, s.rejectStatus = true, http.StatusBadRequest })
	post(t, client, s.URL, largeBody)
	require.Equal(t, largeBody, s.last().body)

	before := s.hitCount()
	post(t, client, s.URL, largeBody)
	require.Equal(t, before+1, s.hitCount(), "a refusal is remembered")
}

func TestTransport_KeepsCompressingWhenThePlainBodyIsABadRequestToo(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	post(t, client, s.URL, largeBody)

	s.set(func(s *server) { s.status = http.StatusBadRequest })
	require.Equal(t, http.StatusBadRequest, postStatus(t, client, s.URL, largeBody))

	s.set(func(s *server) { s.status = 0 })
	post(t, client, s.URL, largeBody)
	require.Equal(t, "gzip", s.last().encoding, "the request was bad, not its encoding")
}

func TestTransport_LeavesAnAlreadyEncodedBodyAlone(t *testing.T) {
	s := newServer(t, "gzip")
	client := &http.Client{Transport: New(http.DefaultTransport)}
	post(t, client, s.URL, largeBody)

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(largeBody))
	_ = zw.Close()
	req, err := http.NewRequest(http.MethodPost, s.URL, bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := client.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.Equal(t, largeBody, s.last().body, "compressed once, not twice")
}

func TestTransport_GivesTheCompressedRequestARereadableBody(t *testing.T) {
	var seen *http.Request
	rt := New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seen = r
		return &http.Response{StatusCode: 200, Header: http.Header{"Accept-Encoding": {"gzip"}}, Body: http.NoBody, Request: r}, nil
	}))
	for i := 0; i < 2; i++ {
		req, err := http.NewRequest(http.MethodPost, "http://rpc.example/", strings.NewReader(largeBody))
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
	}

	require.Equal(t, "gzip", seen.Header.Get("Content-Encoding"))
	again, err := seen.GetBody()
	require.NoError(t, err)
	wire, err := io.ReadAll(again)
	require.NoError(t, err)
	require.Equal(t, seen.ContentLength, int64(len(wire)))
	zr, err := gzip.NewReader(bytes.NewReader(wire))
	require.NoError(t, err)
	body, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.Equal(t, largeBody, string(body))
}

func TestTransport_CompressesGoEthereumRPCCalls(t *testing.T) {
	s := newServer(t, "gzip")
	client, err := rpc.DialOptions(context.Background(), s.URL,
		rpc.WithHTTPClient(&http.Client{Transport: New(http.DefaultTransport)}))
	require.NoError(t, err)
	defer client.Close()
	data := "0x" + strings.Repeat("00000000000000000000000070a08231", 200)

	for i := 0; i < 2; i++ {
		var result string
		require.NoError(t, client.CallContext(context.Background(), &result, "eth_call",
			map[string]string{"to": "0xcA11bde05977b3631167028862bE2a173976CA11", "input": data}, "latest"))
	}

	got := s.last()
	require.Equal(t, "gzip", got.encoding)
	require.Contains(t, got.body, `"method":"eth_call"`)
	require.Contains(t, got.body, data)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
