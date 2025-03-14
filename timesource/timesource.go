package timesource

import (
	"bytes"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
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
var errUpdateOffset = errors.New("failed to compute offset")

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

func computeOffset(timeQuery ntpQuery, servers []string, allowedFailures int) (time.Duration, error) {
	if len(servers) == 0 {
		return 0, nil
	}
	responses := make(chan queryResponse, len(servers))
	for _, server := range servers {
		go func(server string) {
			defer common.LogOnPanic()
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
		collected int
	)
	for response := range responses {
		if response.Error != nil {
			rpcErrors = append(rpcErrors, response.Error)
		} else {
			offsets = append(offsets, response.Offset)
		}
		collected++
		if collected == len(servers) {
			break
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

var defaultTimeSource = &NTPTimeSource{
	servers:           defaultServers,
	allowedFailures:   DefaultMaxAllowedFailures,
	fastNTPSyncPeriod: FastNTPSyncPeriod,
	slowNTPSyncPeriod: SlowNTPSyncPeriod,
	timeQuery:         ntp.QueryWithOptions,
	now:               time.Now,
}

// Default initializes time source with default config values.
func Default() *NTPTimeSource {
	return defaultTimeSource
}

// NTPTimeSource provides source of time that tries to be resistant to time skews.
// It does so by periodically querying time offset from ntp servers.
type NTPTimeSource struct {
	servers           []string
	allowedFailures   int
	fastNTPSyncPeriod time.Duration
	slowNTPSyncPeriod time.Duration
	timeQuery         ntpQuery // for ease of testing
	now               func() time.Time

	quit chan struct{}

	stateMu sync.Mutex
	started bool

	timeDataMu    sync.RWMutex
	latestOffset  time.Duration
	lastMonotonic time.Time
}

// Now returns time adjusted by latest known offset
// and detects system time changes
func (s *NTPTimeSource) Now() time.Time {
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

	// If significant time change detected, update offset synchronously
	if timeDiff.Abs() > TimeChangeThreshold {
		logutils.ZapLogger().Warn("system time change detected",
			zap.Duration("difference", timeDiff),
			zap.Duration("threshold", TimeChangeThreshold))

		// Ignore error as it's logged in updateOffset
		_ = s.updateOffset()

		// Update the reference times only after significant time change
		s.timeDataMu.Lock()
		s.lastMonotonic = s.now()
		s.timeDataMu.Unlock()
	}

	return adjustedTime
}

func (s *NTPTimeSource) updateOffset() error {
	offset, err := computeOffset(s.timeQuery, s.servers, s.allowedFailures)
	if err != nil {
		logutils.ZapLogger().Error("failed to compute offset", zap.Error(err))
		return errUpdateOffset
	}
	logutils.ZapLogger().Info("Difference with ntp servers", zap.Duration("offset", offset))
	s.timeDataMu.Lock()
	defer s.timeDataMu.Unlock()
	s.latestOffset = offset
	//TBD: if we found offset is too large, we should notify user that system time might not be accurate via emit signal,
	// and because go-waku doesn't use NTPTimeSource ATM (it just use time.Now()), this might be a problem for MissingMessageVerifier work normally.
	// e.g. might get errInvalidTimeRange when validate StoreQueryRequest
	return nil
}

// runPeriodically runs periodically the given function based on NTPTimeSource
// synchronization limits (fastNTPSyncPeriod / slowNTPSyncPeriod)
func (s *NTPTimeSource) runPeriodically(fn func() error, starWithSlowSyncPeriod bool) {
	if s.started {
		return
	}

	period := s.fastNTPSyncPeriod
	if starWithSlowSyncPeriod {
		period = s.slowNTPSyncPeriod
	}
	s.quit = make(chan struct{})
	go func() {
		defer common.LogOnPanic()
		for {
			select {
			case <-time.After(period):
				if err := fn(); err == nil {
					period = s.slowNTPSyncPeriod
				} else if period != s.slowNTPSyncPeriod {
					period = s.fastNTPSyncPeriod
				}

			case <-s.quit:
				return
			}
		}
	}()
}

// Start initializes the local offset and starts a goroutine that periodically updates the local offset.
func (s *NTPTimeSource) Start() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.started {
		return
	}

	// Initialize time tracking fields immediately
	currentTime := s.now()
	s.lastMonotonic = currentTime

	// Attempt to update the offset synchronously so that user can have reliable messages right away
	err := s.updateOffset()
	if err != nil {
		// Failure to update can occur if the node is offline.
		// Instead of returning an error, continue with the process as the update will be retried periodically.
		logutils.ZapLogger().Error("failed to update offset", zap.Error(err))
	}

	s.runPeriodically(s.updateOffset, err == nil)

	s.started = true
}

// Stop goroutine that updates time source.
func (s *NTPTimeSource) Stop() error {
	if s.quit == nil {
		return nil
	}
	close(s.quit)
	s.started = false
	return nil
}

func (s *NTPTimeSource) GetCurrentTime() time.Time {
	s.Start()
	return s.Now()
}

func (s *NTPTimeSource) GetCurrentTimeInMillis() uint64 {
	return convertToMillis(s.GetCurrentTime())
}

func GetCurrentTime() time.Time {
	ts := Default()
	ts.Start()
	return ts.Now()
}

func GetCurrentTimeInMillis() uint64 {
	return convertToMillis(GetCurrentTime())
}

func convertToMillis(t time.Time) uint64 {
	return uint64(t.UnixNano() / int64(time.Millisecond))
}
