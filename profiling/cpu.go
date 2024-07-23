package profiling

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"
)

// CPUFilename is a filename in which the CPU profiling is stored.
const CPUFilename = "status_cpu.prof"

var cpuFile *os.File

// StartCPUProfile enables CPU profiling for the current process. While profiling,
// the profile will be buffered and written to the file in folder dataDir.
func StartCPUProfile(dataDir string) error {
	if cpuFile == nil {
		signal.Notify(make(chan os.Signal), syscall.SIGPROF) // enable profiling for shared libraries

		var err error
		cpuFile, err = os.Create(filepath.Join(dataDir, fmt.Sprintf("status_cpu_%d.prof", time.Now().Unix())))
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
	err := cpuFile.Close()
	cpuFile = nil
	return err
}
