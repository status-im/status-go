package profiling

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

// MemFilename is a filename in which the memory profiling is stored.
const MemFilename = "status_mem.prof"

var memFile *os.File

// WriteHeapFile writes heap memory to the file.
func WriteHeapFile(dataDir string) error {
	var err error

	if memFile == nil {
		memFile, err = os.Create(filepath.Join(dataDir, MemFilename))
		if err != nil {
			return err
		}
		defer memFile.Close() //nolint: errcheck
	}
	runtime.GC()
	err = pprof.WriteHeapProfile(memFile)

	return err
}
