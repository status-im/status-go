package timesource

import (
	"sort"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// DefaultServer will be internally resolved to the closest available.
	// also it rarely queries same server more than once.
	DefaultServer = "pool.ntp.org"

	// DefaultAttempts defines how many servers we will query
	DefaultAttempts = 3

	// DefaultUpdatePeriod defines how often time will be queried from ntp.
	DefaultUpdatePeriod = 2 * time.Minute
)

type ntpQuery func(string) (*ntp.Response, error)

func computeOffset(timeQuery ntpQuery, server string, attempts int) (time.Duration, error) {
	offsets := make([]time.Duration, 5)
	// lowest and highest will be cutoff to prevent reading data from out of sync ntp servers
	for i := 0; i < attempts+2; i++ {
		response, err := timeQuery(server)
		if err != nil {
			return 0, err
		}
		offsets[i] = response.ClockOffset
	}
	sort.SliceStable(offsets, func(i, j int) bool {
		return offsets[i] > offsets[j]
	})
	var sum time.Duration
	for i := 1; i <= attempts; i++ {
		sum += offsets[i]
	}
	return sum / time.Duration(attempts), nil
}

// Default initializes time source with default config values.
func Default() *NTPTimeSource {
	return &NTPTimeSource{
		server:       DefaultServer,
		attempts:     DefaultAttempts,
		updatePeriod: DefaultUpdatePeriod,
		timeQuery:    ntp.Query,
	}
}

// NTPTimeSource provides source of time that tries to be resistant to time skews.
// It does so by periodically querying time offset from ntp servers.
type NTPTimeSource struct {
	server       string
	attempts     int
	updatePeriod time.Duration
	timeQuery    ntpQuery // for ease of testing

	quit chan struct{}
	wg   sync.WaitGroup

	mu           sync.RWMutex
	latestOffset time.Duration
}

// Now returns time adjusted by latest known offset
func (s *NTPTimeSource) Now() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().Add(s.latestOffset)
}

func (s *NTPTimeSource) updateOffset() {
	offset, err := computeOffset(s.timeQuery, s.server, s.attempts)
	if err != nil {
		log.Error("failed to compute offset", "error", err)
		return
	}
	log.Info("Difference with ntp servers", "offset", offset)
	s.mu.Lock()
	s.latestOffset = offset
	s.mu.Unlock()
}

// Start runs a goroutine that updates local offset every updatePeriod.
func (s *NTPTimeSource) Start() {
	s.quit = make(chan struct{})
	ticker := time.NewTicker(s.updatePeriod)
	// we try to do it synchronously so that user can have reliable messages right away
	s.updateOffset()
	s.wg.Add(1)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.updateOffset()
			case <-s.quit:
				s.wg.Done()
				return
			}
		}
	}()
}

// Stop goroutine that updates time source.
func (s *NTPTimeSource) Stop() {
	if s.quit == nil {
		return
	}
	close(s.quit)
	s.wg.Wait()
}
