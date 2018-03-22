// +build !metrics

package metrics

import "net/http"

// NewMetricsServer without metrics build flag does not start any metrics server.
func NewMetricsServer(addr string) *http.Server {
	return nil
}
