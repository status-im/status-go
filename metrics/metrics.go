// +build !prometheus

package metrics

import (
	"log"
	"net/http"
)

// StartMetricsServer starts a new HTTP server with expvar handler.
// By default, "/debug/vars" handler is registered.
func StartMetricsServer(addr string) {
	log.Fatal(http.ListenAndServe(addr, nil))
}
