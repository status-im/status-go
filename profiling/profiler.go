package profiling

import (
	"context"
	"fmt"
	"log"
	"net/http"
	hpprof "net/http/pprof"
	"sync"
)

// Profiler runs and controls a HTTP pprof interface.
type Profiler struct {
	mu     sync.Mutex
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
		p.mu.Lock()
		defer p.mu.Unlock()
		log.Printf("debug server stopped: %v", p.server.ListenAndServe())
	}()
}

// Shutdown stops the pprof server.
func (p *Profiler) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.server.Shutdown(ctx)
}
