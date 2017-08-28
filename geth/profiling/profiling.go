package profiling

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

const (
	CPUProfilingFilename = "status_cpu.prof"
	MemProfilingFilename = "status_mem.prof"
)

var (
	cpuFile *os.File
	memFile *os.File
)

// Enables CPU profiling for the current process.
// While profiling, the profile will be buffered and written to file in folder dataDir
func StartCPUProfile(dataDir string) error {
	if cpuFile == nil {
		var err error
		cpuFile, err = os.Create(filepath.Join(dataDir, CPUProfilingFilename))
		if err != nil {
			return err
		}
	}

	return pprof.StartCPUProfile(cpuFile)

}

// Stops the current CPU profile, if any and closes the file
func StopCPUProfile() error {
	if cpuFile == nil {
		return nil
	}
	pprof.StopCPUProfile()
	return cpuFile.Close()
}

// Write heap memory to a file
func WriteHeapFile(dataDir string) error {
	if memFile == nil {
		var err error
		memFile, err = os.Create(filepath.Join(dataDir, MemProfilingFilename))
		if err != nil {
			return err
		}
	}
	runtime.GC()
	return pprof.WriteHeapProfile(memFile)
}
