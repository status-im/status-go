package metrics

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/prometheus"
)

// Server runs and controls a HTTP pprof interface.
type Server struct {
	server *http.Server
}

func NewMetricsServer(port int, r metrics.Registry) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler(r))
	p := Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
	return &p
}

// Listen starts the HTTP server in the background.
func (p *Server) Listen() {
	log.Info("metrics server stopped", "err", p.server.ListenAndServe())
}
