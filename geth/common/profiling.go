package common

import (
	"os"
	"path/filepath"
	"runtime/pprof"
)

const (
	CpuProfilingFilename = "status_cpu.prof"
	MemProfilingFilename = "status_mem.prof"
)

type Profiling struct {
	cpuFile *os.File
	memFile *os.File
}

// Returns new profiling struct
func NewProfiling() *Profiling {
	return &Profiling{}
}

// Enables CPU profiling for the current process.
// While profiling, the profile will be buffered and written to file in folder dataDir
func (p *Profiling) Start(dataDir string) error {
	if err := p.setup(dataDir); err != nil {
		return err
	}
	return pprof.StartCPUProfile(p.cpuFile)
}

// Stops the current CPU profile, if any and closes the file
func (p *Profiling) Stop() error {
	pprof.StopCPUProfile()
	return pprof.WriteHeapProfile(p.memFile)
}

// Setup cpu and mem profiling
func (p *Profiling) setup(dataDir string) error {
	var err error
	p.cpuFile, err = os.Create(filepath.Join(dataDir, CpuProfilingFilename))
	if err != nil {
		return err
	}

	p.memFile, err = os.Create(filepath.Join(dataDir, MemProfilingFilename))
	return err
}
