package storagestats

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/signal"
)

var (
	// ErrNoAccount is returned when no account is logged in.
	ErrNoAccount = errors.New("storagestats: no account is logged in")

	// ErrAlreadyRunning is returned while another collection is in flight.
	ErrAlreadyRunning = errors.New("storagestats: a collection is already running")

	// ErrCancelled is the result error when Stop aborts a collection. Clients
	// treat it as "no result", not as a failure worth showing.
	ErrCancelled = errors.New("cancelled")
)

// Service collects the account load profile on demand. Nothing is collected or
// emitted unless a caller asks.
type Service struct {
	appDB    *sql.DB
	walletDB *sql.DB
	logger   *zap.Logger

	// mu publishes requestID and cancel together: Stop must never observe a
	// started collection whose cancel func is not yet visible, or logout would
	// close sqlcipher under a live COUNT(*). A non-empty requestID is the
	// in-flight guard.
	mu        sync.Mutex
	requestID string
	cancel    context.CancelFunc
}

func New(appDB, walletDB *sql.DB, logger *zap.Logger) *Service {
	return &Service{
		appDB:    appDB,
		walletDB: walletDB,
		logger:   logger.Named("StorageStatsService"),
	}
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "storagestats",
			Version:   "0.1.0",
			Service:   &API{s: s},
		},
	}
}

func (s *Service) Start() error { return nil }

// Stop cancels a collection in flight.
func (s *Service) Stop() error {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

// API is the JSON-RPC surface.
type API struct {
	s *Service
}

// Collect starts a collection and returns the request id the resulting
// storage-stats.progress and storage-stats.result signals will carry. It must
// stay non-blocking: clients call it from their UI thread, and COUNT(*) on a
// sqlcipher table is a full decrypting scan.
func (api *API) Collect(ctx context.Context) (string, error) {
	return api.s.StartCollect()
}

func (s *Service) StartCollect() (string, error) {
	if s.appDB == nil {
		return "", ErrNoAccount
	}

	ctx, cancel := context.WithCancel(context.Background())
	requestID := uuid.New().String()

	s.mu.Lock()
	if s.requestID != "" {
		s.mu.Unlock()
		cancel()
		return "", ErrAlreadyRunning
	}
	s.requestID = requestID
	s.cancel = cancel
	s.mu.Unlock()

	go func() {
		defer panics.LogOnPanic()
		defer cancel()
		// finish releases the guard; this covers a panic before it does.
		defer s.release(requestID)
		s.collect(ctx, requestID)
	}()

	return requestID, nil
}

// release clears the guard unless a later collection already owns it: finish
// releases before emitting the result, so a client may start the next
// collection from inside the signal handler.
func (s *Service) release(requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.requestID != requestID {
		return
	}
	s.requestID = ""
	s.cancel = nil
}

func (s *Service) collect(ctx context.Context, requestID string) {
	profile, err := Collect(ctx, s.appDB, s.walletDB, time.Now(), s.logger,
		func(done, total int) {
			signal.SendStorageStatsProgress(requestID, done, total)
		})

	// Exactly one result per request, cancellation included: clients clear
	// their in-progress state on this signal and on nothing else.
	if ctx.Err() != nil {
		s.finish(requestID, nil, ErrCancelled)
		return
	}
	if err != nil {
		s.logger.Warn("collecting storage stats failed", zap.Error(err))
	}
	s.finish(requestID, profile, err)
}

// finish releases the guard before the result signal, so a client reacting to
// it may collect again immediately.
func (s *Service) finish(requestID string, profile *Profile, err error) {
	s.release(requestID)

	// A nil *Profile inside an interface is not a nil interface: passed
	// straight through it would marshal as "data": null instead of being
	// omitted.
	var data interface{}
	if profile != nil {
		data = profile
	}
	signal.SendStorageStatsResult(requestID, data, err)
}
