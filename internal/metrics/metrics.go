package metrics

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server   *http.Server
	handlers sync.Map
}

func NewMetricsServer(address string) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:              address,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		},
	}

	opts := promhttp.HandlerOpts{}

	// register status handler
	s.RegisterHandler("status", promhttp.HandlerFor(prom.DefaultGatherer, opts))

	mux.Handle("/health", healthHandler())
	mux.Handle("/metrics", s.metricsHandler())

	return s
}

// RegisterHandler adds a new metrics provider with a given name
func (s *Server) RegisterHandler(name string, handler http.Handler) {
	if handler == nil {
		return // Don't store nil handlers
	}
	s.handlers.Store(name, handler)
}

// UnregisterHandler removes a metrics provider
func (s *Server) UnregisterHandler(name string) {
	s.handlers.Delete(name)
}

// metricsHandler creates the combined metrics handler
func (s *Server) metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write all dynamic metrics
		s.handlers.Range(func(key, value interface{}) bool {
			handler := value.(http.Handler) // Safe to cast since we validate in RegisterHandler
			handler.ServeHTTP(w, r)
			return true // continue iteration
		})
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
	err := s.server.ListenAndServe()
	logutils.ZapLogger().Info("metrics server stopped", zap.Error(err))
}

// Stop gracefully shuts down the metrics server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
