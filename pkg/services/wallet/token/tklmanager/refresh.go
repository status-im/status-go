package tklmanager

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/status-im/nim-token-lists/go/tkl"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/pausable"
)

// Write is one list the host persists after a refresh: the core's metadata and
// the bytes the host fetched for it.
type Write struct {
	tkl.ListContent
	Body []byte
}

// RefreshOptions supplies host I/O. Persist, OnSuccess and OnChange run under
// the writer mutex and must not reenter mutating/list methods. OnError runs on
// the scheduler outside that mutex. Persist must durably write the entire batch
// or return an error without writing any of it. Persist and OnSuccess must honor
// context cancellation; Stop cannot preempt a callback that ignores its context.
type RefreshOptions struct {
	Client  *http.Client
	Persist func(context.Context, []Write) error
	// Persist successful checks, including 304s, so restart preserves the next
	// due time. Automatic successes are spaced by RefreshInterval, not CheckInterval.
	OnSuccess       func(context.Context, time.Time) error
	OnChange        func(tkl.Change)
	OnError         func(error)
	Now             func() time.Time
	RefreshInterval time.Duration
	CheckInterval   time.Duration
	MaxBodyBytes    int64
}

type refreshRuntime struct {
	options      RefreshOptions
	client       http.Client
	ctx          context.Context
	stop         context.CancelFunc
	detachParent func() bool
	cancel       context.CancelFunc
	active       bool
	auto         bool
	allowed      bool
	paused       bool
	closing      bool
	done         chan struct{}
	workers      sync.WaitGroup
	pauseCh      chan bool
}

func newRefreshRuntime(options RefreshOptions) (*refreshRuntime, error) {
	if options.Persist == nil {
		return nil, errors.New("refresh persistence is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.RefreshInterval == 0 {
		options.RefreshInterval = 30 * time.Minute
	}
	if options.CheckInterval == 0 {
		options.CheckInterval = 3 * time.Minute
	}
	if options.RefreshInterval < time.Second || options.CheckInterval < time.Second {
		return nil, errors.New("refresh intervals must be at least one second")
	}
	if options.MaxBodyBytes == 0 {
		options.MaxBodyBytes = 16 << 20
	}
	if options.MaxBodyBytes < 1 || options.MaxBodyBytes > 16<<20 {
		return nil, errors.New("invalid HTTP body limit")
	}
	r := &refreshRuntime{options: options, allowed: true, done: make(chan struct{}), pauseCh: make(chan bool, 1)}
	r.ctx, r.stop = context.WithCancel(context.Background())
	if options.Client != nil {
		r.client = *options.Client
	}
	if r.client.Timeout <= 0 || r.client.Timeout > 30*time.Second {
		r.client.Timeout = 30 * time.Second
	}
	redirect := r.client.CheckRedirect
	r.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("token list redirect limit exceeded")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return errors.New("unsupported redirect scheme")
		}
		if redirect != nil {
			return redirect(req, via)
		}
		return nil
	}
	return r, nil
}

func (m *Manager) startRefresh(ctx context.Context, auto bool) error {
	r := m.refreshIO
	if r == nil {
		if auto {
			return ErrRefreshUnavailable
		}
		return nil
	}
	if _, err := m.handle.SetNetworkAllowed(r.allowed); err != nil {
		return err
	}
	if _, err := m.handle.SetAutoRefresh(auto, int64(r.options.RefreshInterval/time.Second), int64(r.options.CheckInterval/time.Second)); err != nil {
		return err
	}
	r.detachParent = context.AfterFunc(ctx, r.stop)
	r.auto = auto
	m.updateTicker()
	r.workers.Add(1)
	go func() {
		defer panics.LogOnPanic()
		defer r.workers.Done()
		tick := func() {
			err := m.refresh(r.ctx, false)
			if err != nil && !errors.Is(err, tkl.Unchanged) && !errors.Is(err, tkl.Busy) && !errors.Is(err, tkl.Aborted) && !errors.Is(err, context.Canceled) {
				if r.options.OnError != nil {
					r.options.OnError(err)
				}
			}
		}
		ticker := pausable.NewPausableTicker(pausable.PausableTickerConfig{Interval: r.options.CheckInterval, OnTick: tick, OnPauseChanged: func(paused bool) {
			if !paused {
				tick()
			}
		}}, r.pauseCh)
		ticker.Run(r.ctx.Done())
	}()
	return nil
}

func (m *Manager) updateTicker() {
	r := m.refreshIO
	// Keep the newest state if HTTP work delays the ticker's consumer.
	select {
	case <-r.pauseCh:
	default:
	}
	r.pauseCh <- r.paused || !r.auto || !r.allowed
}

func (m *Manager) EnableAutoRefresh(ctx context.Context) error  { return m.setAuto(ctx, true) }
func (m *Manager) DisableAutoRefresh(ctx context.Context) error { return m.setAuto(ctx, false) }
func (m *Manager) setAuto(ctx context.Context, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.handle == nil || (m.refreshIO != nil && m.refreshIO.closing) {
		return tkl.Closed
	}
	r := m.refreshIO
	if r == nil {
		if !enabled {
			return nil
		}
		return ErrRefreshUnavailable
	}
	if !m.started {
		return errors.New("catalogue is not started")
	}
	if _, err := m.handle.SetAutoRefresh(enabled, int64(r.options.RefreshInterval/time.Second), int64(r.options.CheckInterval/time.Second)); err != nil {
		return err
	}
	r.auto = enabled
	m.updateTicker()
	return nil
}

// SetNetworkAllowed gates manual and automatic requests independently of the
// auto-refresh preference. Disabling the gate cancels an in-flight request.
func (m *Manager) SetNetworkAllowed(allowed bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.refreshIO
	if m.handle == nil || (r != nil && r.closing) {
		return tkl.Closed
	}
	if r == nil {
		return ErrRefreshUnavailable
	}
	if m.loaded {
		if _, err := m.handle.SetNetworkAllowed(allowed); err != nil {
			return err
		}
	}
	r.allowed = allowed
	if !allowed && r.cancel != nil {
		r.cancel()
	}
	m.updateTicker()
	return nil
}

func (m *Manager) PausableName() string { return "token-lists" }
func (m *Manager) Pause() error         { return m.setPaused(true) }
func (m *Manager) Resume() error        { return m.setPaused(false) }
func (m *Manager) setPaused(paused bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.refreshIO
	if m.handle == nil || (r != nil && r.closing) {
		return tkl.Closed
	}
	if r == nil {
		return nil
	}
	r.paused = paused
	if paused && r.cancel != nil {
		r.cancel()
	}
	m.updateTicker()
	return nil
}
func (m *Manager) PausableState() pausable.ServiceState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.handle == nil || (m.refreshIO != nil && m.refreshIO.closing) {
		return pausable.ServiceStateStopped
	}
	if m.refreshIO != nil && m.refreshIO.paused {
		return pausable.ServiceStatePaused
	}
	return pausable.ServiceStateRunning
}

func (m *Manager) TriggerRefresh(ctx context.Context) error { return m.refresh(ctx, true) }

func (m *Manager) refresh(ctx context.Context, force bool) error {
	m.mu.Lock()
	r := m.refreshIO
	if m.handle == nil || (r != nil && r.closing) {
		m.mu.Unlock()
		return tkl.Closed
	}
	if r == nil {
		m.mu.Unlock()
		return ErrRefreshUnavailable
	}
	if !m.started {
		m.mu.Unlock()
		return errors.New("catalogue is not started")
	}
	if r.paused || !r.allowed {
		m.mu.Unlock()
		return tkl.Aborted
	}
	if r.active {
		m.mu.Unlock()
		return tkl.Busy
	}
	if err := ctx.Err(); err != nil {
		m.mu.Unlock()
		return err
	}
	plan, err := m.handle.RefreshPlan(r.options.Now().Unix(), force)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	callCtx, cancel := context.WithCancel(ctx)
	stopParent := context.AfterFunc(r.ctx, cancel)
	r.cancel = cancel
	r.active = true
	r.workers.Add(1)
	m.mu.Unlock()
	defer func() {
		cancel()
		stopParent()
		m.mu.Lock()
		if m.handle != nil {
			_, _ = m.handle.RefreshAbort(plan.ID, tkl.Aborted)
		}
		r.active = false
		r.cancel = nil
		m.mu.Unlock()
		r.workers.Done()
	}()
	requests := plan.Requests
	// The core keeps no bodies; hold the fetched ones until they are persisted.
	bodies := make(map[string][]byte)
	for {
		results := r.fetchBatch(callCtx, requests)
		for _, result := range results {
			if result.Status == http.StatusOK && result.Failure == nil {
				bodies[result.ID] = result.Body
			}
		}
		m.mu.Lock()
		if err := callCtx.Err(); err != nil {
			m.mu.Unlock()
			return err
		}
		now := r.options.Now().Unix()
		report, err := m.handle.RefreshApply(plan.ID, results, now)
		if err != nil {
			m.mu.Unlock()
			return err
		}
		if report.Step == "NeedMore" {
			requests = report.Requests
			m.mu.Unlock()
			continue
		}
		if report.Step != "Ready" {
			m.mu.Unlock()
			return fmt.Errorf("token refresh failed: %s", report.Outcome)
		}
		writes := make([]Write, len(report.Writes))
		for i, content := range report.Writes {
			body, ok := bodies[content.ID]
			if !ok {
				_, _ = m.handle.RefreshAbort(plan.ID, tkl.StorageFailure)
				m.mu.Unlock()
				return fmt.Errorf("token refresh wrote %s without a fetched body", content.ID)
			}
			writes[i] = Write{ListContent: content, Body: body}
		}
		// The lock protects the prepare/persist/commit interval from all writers.
		// Commit uses this transaction's logical timestamp after durable success.
		if err = r.options.Persist(callCtx, writes); err != nil {
			_, _ = m.handle.RefreshAbort(plan.ID, tkl.StorageFailure)
			m.mu.Unlock()
			return fmt.Errorf("persist token lists: %w", err)
		}
		change, err := m.handle.RefreshCommit(plan.ID, now)
		if err == nil {
			if r.options.OnSuccess != nil {
				err = r.options.OnSuccess(callCtx, time.Unix(now, 0).UTC())
			}
			// The content revision is already durable and visible. A settings
			// error must not suppress its update notification.
			if change.Kind != "NoChange" {
				m.notifyChange(change)
			}
		}
		m.mu.Unlock()
		return err
	}
}

func (m *Manager) notifyChange(change tkl.Change) {
	if m.refreshIO != nil && m.refreshIO.options.OnChange != nil {
		m.refreshIO.options.OnChange(change)
		return
	}
	if m.notify != nil {
		select {
		case m.notify <- struct{}{}:
		default:
		}
	}
}

func (r *refreshRuntime) fetchBatch(ctx context.Context, requests []tkl.FetchRequest) []tkl.FetchResult {
	results := make([]tkl.FetchResult, 0, len(requests))
	// Fetch in bounded windows so request concurrency stays bounded even for a
	// large registry. Bodies travel to the core one at a time, never in an envelope.
	for start := 0; start < len(requests); start += 4 {
		end := min(start+4, len(requests))
		batch := make([]tkl.FetchResult, end-start)
		var pending sync.WaitGroup
		for i := range batch {
			pending.Add(1)
			go func(i int) {
				defer panics.LogOnPanic()
				defer pending.Done()
				batch[i] = r.fetch(ctx, requests[start+i])
			}(i)
		}
		pending.Wait()
		results = append(results, batch...)
	}
	return results
}

func (r *refreshRuntime) fetch(ctx context.Context, request tkl.FetchRequest) tkl.FetchResult {
	result := tkl.FetchResult{ID: request.ID}
	fail := func(err error) tkl.FetchResult {
		result.Failure = &tkl.Diagnostic{Code: "NetworkFailure", Detail: err.Error()}
		return result
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, request.URL, nil)
	if err != nil {
		return fail(err)
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fail(errors.New("unsupported URL scheme"))
	}
	req.Header.Set("User-Agent", "Status-token-lists/0.2")
	if request.ETag != "" {
		req.Header.Set("If-None-Match", request.ETag)
	}
	response, err := r.client.Do(req)
	if err != nil {
		return fail(err)
	}
	defer response.Body.Close()
	result.Status = response.StatusCode
	result.ETag = response.Header.Get("ETag")
	if result.Status != http.StatusOK {
		return result
	}
	var reader io.Reader = response.Body
	if response.Header.Get("Content-Encoding") == "gzip" {
		zipped, err := gzip.NewReader(reader)
		if err != nil {
			return fail(err)
		}
		defer zipped.Close()
		reader = zipped
	} else if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return fail(errors.New("unsupported content encoding"))
	}
	body, err := io.ReadAll(io.LimitReader(reader, r.options.MaxBodyBytes+1))
	if err != nil {
		return fail(err)
	}
	if int64(len(body)) > r.options.MaxBodyBytes {
		result.Failure = &tkl.Diagnostic{Code: "InvalidContent", Detail: "tooLarge"}
		return result
	}
	result.Body = body
	return result
}
