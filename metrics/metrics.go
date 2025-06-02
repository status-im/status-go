package metrics

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/metrics"
	gethprom "github.com/ethereum/go-ethereum/metrics/prometheus"
	"github.com/status-im/status-go/logutils"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/status-im/status-go/common"
)

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server *http.Server
}

func NewMetricsServer(address string, r metrics.Registry) *Server {
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler())
	mux.Handle("/metrics", Handler(r))
	p := Server{
		server: &http.Server{
			Addr:              address,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		},
	}
	return &p
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			logutils.ZapLogger().Error("health handler error", zap.Error(err))
		}
	})
}

func Handler(reg metrics.Registry) http.Handler {
	// we disable compression because geth doesn't support it
	opts := promhttp.HandlerOpts{DisableCompression: true}
	// we are using only our own metrics
	statusMetrics := promhttp.HandlerFor(prom.DefaultGatherer, opts)
	if reg == nil {
		return statusMetrics
	}
	// if registry is provided, combine handlers
	gethMetrics := gethprom.Handler(reg)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusMetrics.ServeHTTP(w, r)
		gethMetrics.ServeHTTP(w, r)
	})
}

// Listen starts the HTTP server in the background.
func (p *Server) Listen() {
	defer common.LogOnPanic()
	logutils.ZapLogger().Info("metrics server stopped", zap.Error(p.server.ListenAndServe()))
}

// Stop gracefully shuts down the metrics server
func (p *Server) Stop() error {
	if p.server != nil {
		return p.server.Close()
	}
	return nil
}
