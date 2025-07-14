package metrics

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ethereum/go-ethereum/metrics"
	gethprom "github.com/ethereum/go-ethereum/metrics/prometheus"
	"github.com/status-im/status-go/logutils"

	"github.com/status-im/status-go/common"
)

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server        *http.Server
	handlers      map[string]http.Handler
	handlersMutex sync.RWMutex
}

func NewMetricsServer(address string, r metrics.Registry) *Server {
	mux := http.NewServeMux()

	s := &Server{
		handlers: make(map[string]http.Handler),
		server: &http.Server{
			Addr:              address,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		},
	}

	// we disable compression because geth doesn't support it
	opts := promhttp.HandlerOpts{DisableCompression: true}
	// register status handler
	s.RegisterHandler("status", promhttp.HandlerFor(prom.DefaultGatherer, opts))

	// register geth handler
	if r != nil {
		s.RegisterHandler("geth", gethprom.Handler(r))
	}

	mux.Handle("/health", healthHandler())
	mux.Handle("/metrics", s.metricsHandler())

	return s
}

// RegisterHandler adds a new metrics provider with a given name
func (s *Server) RegisterHandler(name string, handler http.Handler) {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()
	s.handlers[name] = handler
}

// UnregisterHandler removes a metrics provider
func (s *Server) UnregisterHandler(name string) {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()
	delete(s.handlers, name)
}

// metricsHandler creates the combined metrics handler
func (s *Server) metricsHandler() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Write all dynamic metrics
		s.handlersMutex.RLock()
		for _, handler := range s.handlers {
			if handler != nil {
				handler.ServeHTTP(w, r)
			}
		}
		s.handlersMutex.RUnlock()
	})
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			logutils.ZapLogger().Error("health handler error", zap.Error(err))
		}
	})
}

// Listen starts the HTTP server in the background.
func (s *Server) Listen() {
	defer common.LogOnPanic()
	logutils.ZapLogger().Info("metrics server stopped", zap.Error(s.server.ListenAndServe()))
}

// Stop gracefully shuts down the metrics server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
