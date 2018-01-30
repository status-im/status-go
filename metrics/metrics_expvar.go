// +build metrics,!prometheus

package metrics

import (
	"net/http"
)

// NewMetricsServer starts a new HTTP server with expvar handler.
// By default, "/debug/vars" handler is registered.
func NewMetricsServer(addr string) *http.Server {
	return &http.Server{Addr: addr}
}
