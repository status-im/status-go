package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
)

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server *http.Server
	log    *zap.Logger
}

// NewMetricsServer creates a prometheus server on a particular interface and port
func NewMetricsServer(address string, port int, log *zap.Logger) *Server {
	p := Server{
		log: log.Named("metrics"),
	}

	p.log.Info("starting server", zap.String("address", address), zap.Int("port", port))
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Healthcheck
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	h := &ochttp.Handler{Handler: mux}

	p.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", address, port),
		Handler: h,
	}

	return &p
}

// Start executes the HTTP server in the background.
func (p *Server) Start() {
	p.log.Info("server started ", zap.Error(p.server.ListenAndServe()))
}

// Stop shuts down the prometheus server
func (p *Server) Stop(ctx context.Context) error {
	err := p.server.Shutdown(ctx)
	if err != nil {
		p.log.Error("stopping server", zap.Error(err))
		return err
	}

	return nil
}
