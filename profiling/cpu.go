package profiling

import (
	"os"
	"path/filepath"
	"runtime/pprof"
)

// CPUFilename is a filename in which the CPU profiling is stored.
const CPUFilename = "status_cpu.prof"

var cpuFile *os.File

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
