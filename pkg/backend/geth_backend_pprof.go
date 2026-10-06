package backend

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	pprof "net/http/pprof"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/panics"
)

// pprofServer has its own lock so profiling never waits on StatusBackend.mu.
type pprofServer struct {
	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
}

// StartPprof serves net/http/pprof on addr for on-device profiling. A running server is replaced.
// addr must be a loopback address: the endpoints are unauthenticated.
func (b *StatusBackend) StartPprof(addr string) error {
	if err := requireLoopbackAddr(addr); err != nil {
		return err
	}
	p := &b.pprof
	p.mu.Lock()
	defer p.mu.Unlock()
	_ = p.closeLocked()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	p.server = srv
	p.listener = ln
	go func() {
		defer panics.LogOnPanic()
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			b.logger.Error("pprof server error", zap.Error(err))
		}
	}()
	b.logger.Info("pprof server started", zap.String("addr", ln.Addr().String()))
	return nil
}

// StopPprof shuts down the server started by StartPprof.
func (b *StatusBackend) StopPprof() error {
	p := &b.pprof
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closeLocked()
}

// closeLocked closes the listener too: Server.Close only tracks it once Serve has started.
func (p *pprofServer) closeLocked() error {
	if p.server == nil {
		return nil
	}
	err := p.server.Close()
	if lnErr := p.listener.Close(); lnErr != nil && err == nil && !errors.Is(lnErr, net.ErrClosed) {
		err = lnErr
	}
	p.server = nil
	p.listener = nil
	return err
}

func requireLoopbackAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("pprof address %q is not a loopback address", addr)
}
