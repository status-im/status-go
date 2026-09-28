//go:build tkl

package tklmanager

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/nim-token-lists/go/tkl"
	"github.com/stretchr/testify/require"
)

func TestRefreshPersistsBeforePublication(t *testing.T) {
	for _, settingsFail := range []bool{false, true} {
		t.Run(fmt.Sprint(settingsFail), func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/registry" {
					fmt.Fprintf(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[{"id":"list","sourceUrl":%q}]}`, server.URL+"/list")
					return
				}
				fmt.Fprint(w, `{"name":"Tokens","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000000001","name":"One","symbol":"ONE","decimals":18}]}`)
			}))
			defer server.Close()
			fail := true
			var m *Manager
			var err error
			signals := 0
			observed := make(chan ShadowSnapshot, 3)
			successAttempted := false
			notifiedBeforeSuccess := false
			m, err = New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL + "/registry"}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{
				Persist: func(ctx context.Context, writes []tkl.ListContent) error {
					if len(writes) == 0 {
						return errors.New("expected content writes")
					}
					_, ok := m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
					if ok {
						return errors.New("published before persistence")
					}
					if fail {
						return errors.New("disk full")
					}
					return nil
				}, OnSuccess: func(context.Context, time.Time) error {
					successAttempted = true
					if settingsFail {
						return errors.New("settings failure")
					}
					return nil
				}, OnChange: func(tkl.Change) { notifiedBeforeSuccess = !successAttempted; signals++ },
				OnShadow: func(_ context.Context, s ShadowSnapshot) { observed <- s },
			})
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			require.NoError(t, m.Start(context.Background(), false, nil))
			<-observed
			require.Error(t, m.TriggerRefresh(context.Background()))
			require.Empty(t, observed)
			require.Equal(t, 0, signals)
			require.Equal(t, uint64(1), m.mirror.Load().revision)
			fail = false
			err = m.TriggerRefresh(context.Background())
			if settingsFail {
				require.ErrorContains(t, err, "settings failure")
			} else {
				require.NoError(t, err)
			}
			_, ok := m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
			require.True(t, ok)
			require.Equal(t, 1, signals)
			require.False(t, notifiedBeforeSuccess)
			snapshot := <-observed
			require.Len(t, snapshot.Contents, 2)
			require.Equal(t, m.mirror.Load().revision, snapshot.Revision)
			require.Len(t, snapshot.Tokens, 2)
			for _, row := range snapshot.Contents {
				if row.ID == "list" {
					require.Contains(t, row.Body, `"symbol":"ONE"`)
				}
			}
		})
	}
}

func TestConcurrentStopCancelsPersistence(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
	}))
	defer server.Close()
	var signals atomic.Int32
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(ctx context.Context, _ []tkl.ListContent) error {
		close(entered)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return errors.New("test released persistence")
		}
	}, OnChange: func(tkl.Change) { signals.Add(1) }})
	require.NoError(t, err)
	require.NoError(t, m.Start(context.Background(), false, nil))
	refreshed := make(chan error, 1)
	go func() { refreshed <- m.TriggerRefresh(context.Background()) }()
	<-entered
	stopped := make(chan error, 2)
	go func() { stopped <- m.Stop() }()
	go func() { stopped <- m.Stop() }()
	// Release the fake on failure so the regression cannot strand test workers.
	defer close(release)
	for i := 0; i < 2; i++ {
		select {
		case err := <-stopped:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Error("Stop did not cancel persistence")
			return
		}
	}
	require.ErrorIs(t, <-refreshed, context.Canceled)
	require.Zero(t, signals.Load())
}

func TestReadOnlyPauseIsNoOp(t *testing.T) {
	m, err := New(tkl.Config{Chains: []uint64{1}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil })
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	require.NoError(t, m.Pause())
	require.NoError(t, m.Resume())
}

func TestStopCancelsSuccessWrite(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
	}))
	defer server.Close()
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnSuccess: func(ctx context.Context, _ time.Time) error {
		close(entered)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return errors.New("released")
		}
	}})
	require.NoError(t, err)
	require.NoError(t, m.Start(context.Background(), false, nil))
	refreshed := make(chan error, 1)
	go func() { refreshed <- m.TriggerRefresh(context.Background()) }()
	<-entered
	stopped := make(chan error, 1)
	go func() { stopped <- m.Stop() }()
	defer close(release)
	select {
	case err := <-stopped:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Error("Stop did not cancel settings write")
		return
	}
	require.ErrorIs(t, <-refreshed, context.Canceled)
}

func TestFetchBatchBoundedConcurrency(t *testing.T) {
	entered := make(chan struct{}, 8)
	release := make(chan struct{})
	defer close(release)
	var active, maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := maximum.Load(); n > old; old = maximum.Load() {
			if maximum.CompareAndSwap(old, n) {
				break
			}
		}
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
		fmt.Fprint(w, r.URL.Path)
	}))
	defer server.Close()
	runtime, err := newRefreshRuntime(RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }})
	require.NoError(t, err)
	defer runtime.stop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requests := make([]tkl.FetchRequest, 8)
	for i := range requests {
		requests[i] = tkl.FetchRequest{ID: fmt.Sprint(i), URL: fmt.Sprintf("%s/%d", server.URL, i)}
	}
	done := make(chan []tkl.FetchResult, 1)
	go func() { done <- runtime.fetchBatch(ctx, requests) }()
	for i := 0; i < 4; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Error("requests did not overlap")
			return
		}
	}
	cancel() // Cancellation must drain the whole bounded batch.
	select {
	case results := <-done:
		require.Len(t, results, 8)
		for i, result := range results {
			require.Equal(t, requests[i].ID, result.ID)
		}
	case <-time.After(time.Second):
		t.Error("batch did not drain")
	}
	require.Equal(t, int32(4), maximum.Load())
}

func TestAutomaticRefreshPausesAndResumes(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
	}))
	defer server.Close()
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{RefreshInterval: time.Second, CheckInterval: time.Second, Persist: func(context.Context, []tkl.ListContent) error { return nil }})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), true, nil))
	require.Eventually(t, func() bool { return calls.Load() > 0 }, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, m.Pause())
	before := calls.Load()
	time.Sleep(1200 * time.Millisecond)
	require.Equal(t, before, calls.Load())
	require.ErrorIs(t, m.TriggerRefresh(context.Background()), tkl.Aborted)
	require.NoError(t, m.Resume())
	require.Eventually(t, func() bool { return calls.Load() > before }, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, m.DisableAutoRefresh(context.Background()))
	// A manual request still works and does not restart automatic scheduling.
	require.Eventually(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return !m.refreshIO.active }, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, m.TriggerRefresh(context.Background()))
	before = calls.Load()
	time.Sleep(1200 * time.Millisecond)
	require.Equal(t, before, calls.Load())
	require.NoError(t, m.Stop())
	require.NoError(t, m.Stop())
}

func TestPartialRefreshRetainsInvalidSource(t *testing.T) {
	const body = `{"name":"List","timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000000001","name":"One","symbol":"ONE","decimals":18}]}`
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/registry":
			fmt.Fprintf(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[{"id":"good","sourceUrl":%q},{"id":"bad","sourceUrl":%q}]}`, server.URL+"/good", server.URL+"/bad")
		case "/good":
			fmt.Fprint(w, body)
		default:
			fmt.Fprint(w, "malformed")
		}
	}))
	defer server.Close()
	initial := strings.ReplaceAll(strings.ReplaceAll(body, "0000000001", "0000000002"), "ONE", "OLD")
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL + "/registry", InitialLists: []tkl.ListContent{{ID: "bad", Format: tkl.StandardFormat, Body: initial}}}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(_ context.Context, writes []tkl.ListContent) error {
		for _, row := range writes {
			if row.ID == "bad" {
				return errors.New("invalid source was persisted")
			}
		}
		return nil
	}})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	require.NoError(t, m.TriggerRefresh(context.Background()))
	token, ok := m.GetTokenByChainAddress(1, common.HexToAddress("0x2"))
	require.True(t, ok)
	require.Equal(t, "OLD", token.Symbol)
	_, ok = m.GetTokenByChainAddress(1, common.HexToAddress("0x1"))
	require.True(t, ok)
	state, err := m.handle.RefreshState()
	require.NoError(t, err)
	require.Equal(t, "Partial", state.LastOutcome)
}

func TestRefreshCancellationAndSupersession(t *testing.T) {
	for _, action := range []string{"pause", "stop", "chains", "network"} {
		t.Run(action, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				select {
				case <-r.Context().Done():
				case <-release:
				}
				fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
			}))
			defer server.Close()
			persists := 0
			m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { persists++; return nil }})
			require.NoError(t, err)
			defer func() { _ = m.Stop() }()
			require.NoError(t, m.Start(context.Background(), false, nil))
			result := make(chan error, 1)
			go func() { result <- m.TriggerRefresh(context.Background()) }()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("no request")
			}
			switch action {
			case "pause":
				require.NoError(t, m.Pause())
			case "stop":
				require.NoError(t, m.Stop())
			case "chains":
				require.NoError(t, m.SetChains([]uint64{10}))
			case "network":
				require.NoError(t, m.SetNetworkAllowed(false))
			}
			close(release)
			select {
			case err := <-result:
				require.Error(t, err)
			case <-time.After(3 * time.Second):
				t.Fatal("refresh did not drain")
			}
			require.Equal(t, 0, persists)
		})
	}
}

func TestHTTPBounds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Status-token-lists/0.2", r.Header.Get("User-Agent"))
		switch r.URL.Path {
		case "/gzip":
			w.Header().Set("Content-Encoding", "gzip")
			z := gzip.NewWriter(w)
			_, _ = z.Write([]byte(strings.Repeat("x", 65)))
			_ = z.Close()
		case "/redirect":
			http.Redirect(w, r, "/redirect", http.StatusFound)
		case "/etag":
			require.Equal(t, "v1", r.Header.Get("If-None-Match"))
			w.WriteHeader(http.StatusNotModified)
		case "/slow":
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
		default:
			fmt.Fprint(w, "valid")
		}
	}))
	defer server.Close()
	r, err := newRefreshRuntime(RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, MaxBodyBytes: 64, Client: &http.Client{Timeout: 50 * time.Millisecond}})
	require.NoError(t, err)
	for _, path := range []string{"/gzip", "/redirect", "/slow"} {
		result := r.fetch(context.Background(), tkl.FetchRequest{ID: "x", URL: server.URL + path})
		require.NotNil(t, result.Failure, path)
	}
	result := r.fetch(context.Background(), tkl.FetchRequest{ID: "x", URL: server.URL + "/etag", ETag: "v1"})
	require.Nil(t, result.Failure)
	require.Equal(t, 304, result.Status)
	result = r.fetch(context.Background(), tkl.FetchRequest{ID: "x", URL: server.URL})
	require.Equal(t, "valid", result.Body)
}

func TestUnchangedRefreshDoesNotSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == "v1" {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", "v1")
		fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
	}))
	defer server.Close()
	signals, successes := 0, 0
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnSuccess: func(context.Context, time.Time) error { successes++; return nil }, OnChange: func(tkl.Change) { signals++ }})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	require.NoError(t, m.TriggerRefresh(context.Background()))
	first := signals
	require.NoError(t, m.TriggerRefresh(context.Background()))
	require.Equal(t, first, signals)
	require.Equal(t, 2, successes)
}

func TestSuccessful304UsesRefreshIntervalNotCheckInterval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == "v1" {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", "v1")
		fmt.Fprint(w, `{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[]}`)
	}))
	defer server.Close()
	now := int64(100)
	successes := 0
	m, err := New(tkl.Config{Chains: []uint64{1}, RegistryID: "registry", RegistryURL: server.URL}, func(context.Context) (tkl.Bootstrap, error) { return tkl.Bootstrap{}, nil }, RefreshOptions{Now: func() time.Time { return time.Unix(now, 0) }, Persist: func(context.Context, []tkl.ListContent) error { return nil }, OnSuccess: func(context.Context, time.Time) error { successes++; return nil }})
	require.NoError(t, err)
	defer func() { _ = m.Stop() }()
	require.NoError(t, m.Start(context.Background(), false, nil))
	// Drive the scheduler's non-forced path with an explicit clock, without a
	// wall-clock ticker racing this table of due times.
	_, err = m.handle.SetAutoRefresh(true, 1800, 180)
	require.NoError(t, err)
	require.NoError(t, m.refresh(context.Background(), false))
	now = 280
	require.ErrorIs(t, m.refresh(context.Background(), false), tkl.Unchanged)
	require.Equal(t, 1, successes)
	now = 1900
	require.NoError(t, m.refresh(context.Background(), false))
	require.Equal(t, 2, successes)
	now = 2080
	require.ErrorIs(t, m.refresh(context.Background(), false), tkl.Unchanged)
	require.Equal(t, 2, successes)
}
