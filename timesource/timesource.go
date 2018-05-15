package timesource

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// DefaultMaxAllowedFailures defines how many failures will be tolerated.
	DefaultMaxAllowedFailures = 2

	// DefaultUpdatePeriod defines how often time will be queried from ntp.
	DefaultUpdatePeriod = 2 * time.Minute

	// DefaultRPCTimeout defines write deadline for single ntp server request.
	DefaultRPCTimeout = 2 * time.Second
)

// defaultServers will be resolved to the closest available,
// and with high probability resolved to the different IPs
var defaultServers = []string{
	"0.pool.ntp.org",
	"1.pool.ntp.org",
	"2.pool.ntp.org",
	"3.pool.ntp.org",
}

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
			response, err := timeQuery(server, ntp.QueryOptions{
				Timeout: DefaultRPCTimeout,
			})
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

// Default initializes time source with default config values.
func Default() *NTPTimeSource {
	return &NTPTimeSource{
		servers:         defaultServers,
		allowedFailures: DefaultMaxAllowedFailures,
		updatePeriod:    DefaultUpdatePeriod,
		timeQuery:       ntp.QueryWithOptions,
	}
}

// NTPTimeSource provides source of time that tries to be resistant to time skews.
// It does so by periodically querying time offset from ntp servers.
type NTPTimeSource struct {
	servers         []string
	allowedFailures int
	updatePeriod    time.Duration
	timeQuery       ntpQuery // for ease of testing

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
	offset, err := computeOffset(s.timeQuery, s.servers, s.allowedFailures)
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
func (s *NTPTimeSource) Start(*p2p.Server) error {
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
	return nil
}

// Stop goroutine that updates time source.
func (s *NTPTimeSource) Stop() error {
	if s.quit == nil {
		return nil
	}
	close(s.quit)
	s.wg.Wait()
	return nil
}

// APIs used to be conformant with service interface
func (s *NTPTimeSource) APIs() []rpc.API {
	return nil
}

// Protocols used to conformant with service interface
func (s *NTPTimeSource) Protocols() []p2p.Protocol {
	return nil
}
