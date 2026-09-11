package httpbytes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestWrapCountsRequestAndResponseByHost(t *testing.T) {
	const responseBody = "token-list-payload"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "rpc-body", string(body))
		_, err = io.WriteString(w, responseBody)
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	host := stripPort(strings.TrimPrefix(server.URL, "http://"))
	before := bytesForHost(t, host)

	client := &http.Client{Transport: Wrap(http.DefaultTransport)}
	resp, err := client.Post(server.URL+"/tokens", "text/plain", strings.NewReader("rpc-body"))
	require.NoError(t, err)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, responseBody, string(got))

	after := bytesForHost(t, host)
	require.Equal(t, before.in+len(responseBody), after.in)
	require.Equal(t, before.out+len("rpc-body"), after.out)
}

func TestWrapIsIdempotent(t *testing.T) {
	once := Wrap(http.DefaultTransport)
	require.Equal(t, once, Wrap(once))
}

func TestHostLabelStripsPort(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://prod.market.status.im:443/tokens", nil)
	require.NoError(t, err)
	require.Equal(t, "prod.market.status.im", HostLabel(req))
}

func TestAddResponseBytesCountsInboundBytes(t *testing.T) {
	host := "prod.market.status.im"
	before := bytesForHost(t, host)
	AddResponseBytes(host, 21)
	after := bytesForHost(t, host)
	require.Equal(t, before.in+21, after.in)
}

func TestHostFromURL(t *testing.T) {
	require.Equal(t, "prod.market.status.im", HostFromURL("https://prod.market.status.im/static/lists.json"))
	require.Equal(t, unknownHost, HostFromURL("not a url"))
}

func TestWrapClientIsIdempotentOnTransport(t *testing.T) {
	client := WrapClient(&http.Client{Transport: http.DefaultTransport})
	require.Equal(t, client.Transport, WrapClient(client).Transport)
}

type hostBytes struct {
	in  int
	out int
}

func bytesForHost(t *testing.T, host string) hostBytes {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	var got hostBytes
	for _, family := range families {
		if family.GetName() != "status_http_bytes" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := labelMap(metric)
			if labels["host"] != host {
				continue
			}
			value := int(metric.GetCounter().GetValue())
			switch labels["dir"] {
			case dirIn:
				got.in = value
			case dirOut:
				got.out = value
			}
		}
	}
	return got
}

func labelMap(metric *dto.Metric) map[string]string {
	labels := make(map[string]string, len(metric.GetLabel()))
	for _, label := range metric.GetLabel() {
		labels[label.GetName()] = label.GetValue()
	}
	return labels
}
