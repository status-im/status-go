package profiling

import (
	"net/http"
	hpprof "net/http/pprof"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

// Profiler runs and controls a HTTP pprof interface.
type Profiler struct {
	server *http.Server
}

// NewProfiler creates an instance of the profiler with
// the given port.
func NewProfiler(address string) *Profiler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", hpprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", hpprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", hpprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", hpprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", hpprof.Trace)
	p := Profiler{
		server: &http.Server{
			Addr:              address,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		},
	}
	return &p
}

// Go starts the HTTP pprof in the background.
func (p *Profiler) Go() {
	go func() {
		defer common.LogOnPanic()
		logutils.ZapLogger().Info("debug server stopped", zap.Error(p.server.ListenAndServe()))
	}()
	logutils.ZapLogger().Info("debug server started")
}
