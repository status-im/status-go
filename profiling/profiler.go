package profiling

import (
	"context"
	"fmt"
	"net/http"
	hpprof "net/http/pprof"

	"github.com/ethereum/go-ethereum/log"
)

// Profiler runs and controls a HTTP pprof interface.
type Profiler struct {
	server *http.Server
}

// NewProfiler creates an instance of the profiler with
// the given port.
func NewProfiler(port int) *Profiler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", hpprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", hpprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", hpprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", hpprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", hpprof.Trace)
	p := Profiler{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
	return &p
}

// Run starts the HTTP pprof in the background.
func (p *Profiler) Run() {
	go func() {
		log.Info("debug server stopped", "err", p.server.ListenAndServe())
	}()
	log.Info("debug server started")
}

// Shutdown stops the pprof server.
func (p *Profiler) Shutdown(ctx context.Context) error {
	return p.server.Shutdown(ctx)
}
