package common

import (
	"os"
	"path/filepath"
	"runtime/pprof"
)

const (
	PROFILING_FILENAME = "status.prof"
)

type Profiling struct {
	file *os.File
}

// Returns new profiling struct
func NewProfiling() *Profiling {
	return &Profiling{}
}

// Enables CPU profiling for the current process.
// While profiling, the profile will be buffered and written to file in folder dataDir
func (p *Profiling) Start(dataDir string) *Profiling {
	if p.file == nil {
		p.file, _ = os.Create(filepath.Join(dataDir, PROFILING_FILENAME))

	}
	pprof.StartCPUProfile(p.file)
	return p
}

// Stops the current CPU profile, if any and closes the file
func (p *Profiling) Stop() *Profiling {
	if p.file == nil {
		return p
	}
	pprof.StopCPUProfile()
	p.file.Close()
	return p
}
