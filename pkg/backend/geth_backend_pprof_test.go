package backend

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/testutils"
)

func TestStartPprofServesHeapProfile(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	port := reserveLocalhostPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	require.NoError(t, b.StartPprof(addr))
	t.Cleanup(func() { _ = b.StopPprof() })

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + addr + "/debug/pprof/heap?debug=1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "heap profile")
}

func TestStartPprofReturnsErrorWhenAddressInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	b := NewStatusBackend(testutils.MustCreateTestLogger())
	require.Error(t, b.StartPprof(ln.Addr().String()))
	require.NoError(t, b.StopPprof())
}

func TestStopPprofClosesServerAndIsIdempotent(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	port := reserveLocalhostPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	require.NoError(t, b.StartPprof(addr))
	require.NoError(t, b.StartPprof(addr))
	require.NoError(t, b.StopPprof())
	require.NoError(t, b.StopPprof())

	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://" + addr + "/debug/pprof/")
	require.Error(t, err)
}
