package timesource

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/beevik/ntp"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/pausable"
)

const (
	// DefaultMaxAllowedFailures defines how many failures will be tolerated.
	DefaultMaxAllowedFailures = 4

	// FastNTPSyncPeriod period between ntp synchronizations before the first
	// successful connection.
	FastNTPSyncPeriod = 2 * time.Minute

	// SlowNTPSyncPeriod period between ntp synchronizations after the first
	// successful connection.
	SlowNTPSyncPeriod = 1 * time.Hour

	// DefaultRPCTimeout defines write deadline for single ntp server request.
	DefaultRPCTimeout = 2 * time.Second

	// DefaultQueryDeadline bounds the wait for all servers, name resolution included.
	DefaultQueryDeadline = 2 * DefaultRPCTimeout

	// TimeChangeThreshold defines the minimum time difference that indicates
	// system time has been changed. Values smaller than this are considered normal drift.
	TimeChangeThreshold = 1 * time.Second
)

// defaultServers will be resolved to the closest available,
// and with high probability resolved to the different IPs
var defaultServers = []string{
	"time.apple.com",
	"pool.ntp.org",
	"time.cloudflare.com",
	"time.windows.com",
	"ntp.neu.edu.cn",
	"ntp.nict.jp",
	"amazon.pool.ntp.org",
	"android.pool.ntp.org",
}
var (
	errUpdateOffset  = errors.New("failed to compute offset")
	errQueryDeadline = errors.New("ntp server did not answer before the deadline")
)

type ntpQuery func(string, ntp.QueryOptions) (*ntp.Response, error)

type queryResponse struct {
	Offset time.Duration
	Error  error
}

type multiRPCError []error

func (e multiRPCError) Error() string {
	var b bytes.Buffer
	b.WriteString("RPC failed: ")
	more := false
	for _, err := range e {
		if more {
			b.WriteString("; ")
		}
		b.WriteString(err.Error())
		more = true
	}
	b.WriteString(".")
	return b.String()
}

func computeOffset(timeQuery ntpQuery, servers []string, allowedFailures int, deadline time.Duration) (time.Duration, error) {
	if len(servers) == 0 {
		return 0, nil
	}
	responses := make(chan queryResponse, len(servers))
	for _, server := range servers {
		go func(server string) {
			defer panics.LogOnPanic()
			response, err := timeQuery(server, ntp.QueryOptions{
				Timeout: DefaultRPCTimeout,
			})
			if err == nil {
				err = response.Validate()
			}
			if err != nil {
				responses <- queryResponse{Error: err}
				return
			}
			responses <- queryResponse{Offset: response.ClockOffset}
		}(server)
	}
	var (
		rpcErrors multiRPCError
		offsets   []time.Duration
	)
	timeout := time.NewTimer(deadline)
	defer timeout.Stop()
	for collected := 0; collected < len(servers); collected++ {
		select {
		case response := <-responses:
			if response.Error != nil {
				rpcErrors = append(rpcErrors, response.Error)
			} else {
				offsets = append(offsets, response.Offset)
			}
		case <-timeout.C:
			for ; collected < len(servers); collected++ {
				rpcErrors = append(rpcErrors, errQueryDeadline)
			}
		}
	}
	if lth := len(rpcErrors); lth > allowedFailures {
		return 0, rpcErrors
	} else if lth == len(servers) {
		return 0, rpcErrors
	}
	sort.SliceStable(offsets, func(i, j int) bool {
		return offsets[i] > offsets[j]
	})
	mid := len(offsets) / 2
	if len(offsets)%2 == 0 {
		return (offsets[mid-1] + offsets[mid]) / 2, nil
	}
	return offsets[mid], nil
}

var defaultNTPTimeSource = &ntpTimeSource{
	servers:           defaultServers,
	allowedFailures:   DefaultMaxAllowedFailures,
	fastNTPSyncPeriod: FastNTPSyncPeriod,
	slowNTPSyncPeriod: SlowNTPSyncPeriod,
	timeQuery:         ntp.QueryWithOptions,
	now:               time.Now,
	fastRetry:         make(chan struct{}, 1),
}

// ntpTimeSource provides source of time that tries to be resistant to time skews.
// It does so by periodically querying time offset from ntp servers.
type ntpTimeSource struct {
	pausable.PauseBroadcaster

	servers           []string
	allowedFailures   int
	fastNTPSyncPeriod time.Duration
	slowNTPSyncPeriod time.Duration
	timeQuery         ntpQuery // for ease of testing
	now               func() time.Time

	stateMu sync.Mutex
	cancel  context.CancelFunc
	started bool

	timeDataMu    sync.RWMutex
	latestOffset  time.Duration
	lastMonotonic time.Time

	offsetUpdating atomic.Bool
	// fastRetry asks the periodic sync to retry at the fast period.
	fastRetry chan struct{}
}

// Now returns time adjusted by latest known offset
// and detects system time changes
func (s *ntpTimeSource) Now() time.Time {
	s.timeDataMu.RLock()

	currentTime := s.now()
	adjustedTime := currentTime.Add(s.latestOffset)

	// Skip time change detection if time tracking not initialized yet
	if s.lastMonotonic.IsZero() {
		s.timeDataMu.RUnlock()
		return adjustedTime
	}

	// Check for time inconsistency
	monotonicElapsed := time.Since(s.lastMonotonic)
	wallClockElapsed := time.Duration(currentTime.UnixNano() - s.lastMonotonic.UnixNano())
	timeDiff := monotonicElapsed - wallClockElapsed

	s.timeDataMu.RUnlock()

	if timeDiff.Abs() > TimeChangeThreshold {
		logutils.ZapLogger().Warn("system time change detected",
			zap.Duration("difference", timeDiff),
			zap.Duration("threshold", TimeChangeThreshold))

		s.requestOffsetUpdate()

		// Update the reference times only after significant time change
		s.timeDataMu.Lock()
		s.lastMonotonic = s.now()
		s.timeDataMu.Unlock()
	}

	return adjustedTime
}

// requestOffsetUpdate updates the offset in the background, unless an update is already running.
func (s *ntpTimeSource) requestOffsetUpdate() {
	if !s.offsetUpdating.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer panics.LogOnPanic()
		defer s.offsetUpdating.Store(false)
		if err := s.updateOffset(); err != nil {
			select {
			case s.fastRetry <- struct{}{}:
			default:
			}
		}
	}()
}

func (s *ntpTimeSource) updateOffset() error {
	offset, err := computeOffset(s.timeQuery, s.servers, s.allowedFailures, DefaultQueryDeadline)
	if err != nil {
		logutils.ZapLogger().Error("failed to compute offset", zap.Error(err))
		return errUpdateOffset
	}
	logutils.ZapLogger().Info("Difference with ntp servers", zap.Duration("offset", offset))
	s.timeDataMu.Lock()
	defer s.timeDataMu.Unlock()
	s.latestOffset = offset
	// The fresh offset covers any drift so far, so measure clock changes from here.
	if !s.lastMonotonic.IsZero() {
		s.lastMonotonic = s.now()
	}
	//TBD: if we found offset is too large, we should notify user that system time might not be accurate via emit signal,
	// and because go-waku doesn't use ntpTimeSource ATM (it just use time.Now()), this might be a problem for MissingMessageVerifier work normally.
	// e.g. might get errInvalidTimeRange when validate StoreQueryRequest
	return nil
}

// runPeriodically runs periodically the given function based on ntpTimeSource
// synchronization limits (fastNTPSyncPeriod / slowNTPSyncPeriod)
func (s *ntpTimeSource) runPeriodically(ctx context.Context, fn func() error, starWithSlowSyncPeriod bool) {
	period := s.fastNTPSyncPeriod
	if starWithSlowSyncPeriod {
		period = s.slowNTPSyncPeriod
	}
	go func() {
		defer panics.LogOnPanic()
		sub := s.Subscribe()
		defer sub.Unsubscribe()

		paused := <-sub.C()
		if paused && period != s.slowNTPSyncPeriod {
			period = s.slowNTPSyncPeriod
		}

		timer := time.NewTimer(period)
		defer timer.Stop()

		resetTimer := func(d time.Duration) {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(d)
		}

		for {
			select {
			case pausedState, ok := <-sub.C():
				if !ok {
					return
				}
				paused = pausedState
				if paused && period != s.slowNTPSyncPeriod {
					period = s.slowNTPSyncPeriod
				}
				resetTimer(period)
			case <-s.fastRetry:
				if !paused {
					period = s.fastNTPSyncPeriod
					resetTimer(period)
				}
			case <-timer.C:
				if paused {
					resetTimer(period)
					continue
				}
				if err := fn(); err == nil {
					period = s.slowNTPSyncPeriod
				} else if period != s.slowNTPSyncPeriod {
					period = s.fastNTPSyncPeriod
				}
				if paused && period != s.slowNTPSyncPeriod {
					period = s.slowNTPSyncPeriod
				}
				resetTimer(period)

			case <-ctx.Done():
				return
			}
		}
	}()
}

// Start initializes the local offset and starts a goroutine that periodically updates the local offset.
func (s *ntpTimeSource) Start(ctx context.Context) error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.started {
		return nil
	}

	// Initialize time tracking fields immediately
	s.timeDataMu.Lock()
	s.lastMonotonic = s.now()
	s.timeDataMu.Unlock()
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.started = true

	go func() {
		defer panics.LogOnPanic()
		err := s.updateOffset()
		if err != nil {
			// Failure to update can occur if the node is offline.
			// Instead of returning an error, continue with the process as the update will be retried periodically.
			logutils.ZapLogger().Error("failed to update offset", zap.Error(err))
		}
		s.runPeriodically(ctx, s.updateOffset, err == nil)
	}()

	return nil
}

// Stop goroutine that updates time source.
func (s *ntpTimeSource) Stop() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.cancel == nil {
		return
	}
	s.cancel()
	s.started = false
}

func (s *ntpTimeSource) GetCurrentTime() time.Time {
	err := s.Start(context.Background())
	if err != nil {
		panic("could not obtain timesource: " + err.Error())
	}
	return s.Now()
}

func (s *ntpTimeSource) GetCurrentTimeInMillis() uint64 {
	return convertToMillis(s.GetCurrentTime())
}

func GetCurrentTime() time.Time {
	ts := DefaultService()
	err := ts.Start(context.Background())
	if err != nil {
		panic("could not obtain timesource: " + err.Error())
	}
	return ts.Now()
}

func GetCurrentTimeInMillis() uint64 {
	return convertToMillis(GetCurrentTime())
}

func convertToMillis(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}
