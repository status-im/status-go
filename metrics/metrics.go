package metrics

import (
	"fmt"
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

func NewMetricsServer(port int, r metrics.Registry) *Server {
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler())
	mux.Handle("/metrics", Handler(r))
	p := Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
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
	// we are combining handlers to avoid having 2 endpoints
	statusMetrics := promhttp.HandlerFor(prom.DefaultGatherer, opts) // our metrics
	gethMetrics := gethprom.Handler(reg)                             // geth metrics
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
