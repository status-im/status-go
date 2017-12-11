package profiling

import (
	"context"
	"fmt"
	"log"
	"net/http"
	hpprof "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
)

const (
	// CPUFilename is a filename in which the CPU profiling is stored.
	CPUFilename = "status_cpu.prof"
	// MemFilename is a filename in which the memory profiling is stored.
	MemFilename = "status_mem.prof"
)

var (
	cpuFile *os.File
	memFile *os.File
)

// StartCPUProfile enables CPU profiling for the current process. While profiling,
// the profile will be buffered and written to the file in folder dataDir.
func StartCPUProfile(dataDir string) error {
	if cpuFile == nil {
		var err error
		cpuFile, err = os.Create(filepath.Join(dataDir, CPUFilename))
		if err != nil {
			return err
		}
	}

	return pprof.StartCPUProfile(cpuFile)
}

// StopCPUProfile stops the current CPU profile, if any, and closes the file.
func StopCPUProfile() error {
	if cpuFile == nil {
		return nil
	}
	pprof.StopCPUProfile()
	return cpuFile.Close()
}

// WriteHeapFile writes heap memory to the file.
func WriteHeapFile(dataDir string) error {
	if memFile == nil {
		var err error
		memFile, err = os.Create(filepath.Join(dataDir, MemFilename))
		if err != nil {
			return err
		}
		defer memFile.Close()
	}
	runtime.GC()
	return pprof.WriteHeapProfile(memFile)
}

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
