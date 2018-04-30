package timesource

import (
	"bytes"
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

func computeOffset(timeQuery ntpQuery, server string, attempts int) (time.Duration, error) {
	responses := make(chan queryResponse, attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			// ntp.Query default timeout is 5s
			response, err := timeQuery(server)
			if err != nil {
				responses <- queryResponse{Error: err}
				return
			}
			responses <- queryResponse{Offset: response.ClockOffset}
		}()
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
		if collected == attempts {
			break
		}
	}
	if len(rpcErrors) != 0 {
		return 0, rpcErrors
	}
	sort.SliceStable(offsets, func(i, j int) bool {
		return offsets[i] > offsets[j]
	})
	return offsets[attempts/2], nil
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
