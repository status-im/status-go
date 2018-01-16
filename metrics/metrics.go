// +build !prometheus

package metrics

import (
	"expvar"
	"log"
	"net/http"
)

// StartMetricsServer starts a new HTTP server with expvar handler.
func StartMetricsServer(addr string) {
	http.Handle("/debug/vars", expvar.Handler())
	log.Fatal(http.ListenAndServe(addr, nil))
}
