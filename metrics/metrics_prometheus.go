// +build metrics,prometheus

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer starts a new HTTP server with prometheus handler.
func NewMetricsServer(addr string) *http.Server {
	server := http.Server{Addr: addr}
	http.Handle("/metrics", promhttp.Handler())

	return &server
}
