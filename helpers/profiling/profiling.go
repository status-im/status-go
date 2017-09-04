package profiling

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
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
