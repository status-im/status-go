package profiling

import (
	"os"
	"testing"
	"time"
)

func TestProfiling(t *testing.T) {
	if err := StartCPUProfile(""); err != nil {
		t.Error("Profiling start return error:", err)
		return
	}

	// Some blocking task to collect data
	time.Sleep(3 * time.Second)

	if err := StopCPUProfile(); err != nil {
		t.Error("Profiling stop return error:", err)
		return
	}

	//check if the file is present on system
	file, err := os.Open(CPUProfilingFilename)
	if err != nil {
		t.Error("Profiling not found on filesystem:", err)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		t.Error("Cannot get file information for cpu profile", err)
		return
	}

	if info.Size() == 0 {
		t.Error("Empty CPU profiling file", err)
		return
	}

	if err := WriteHeapFile(""); err != nil {
		t.Error("Cannot write heap file", err)
		return
	}

	file, err = os.Open(MemProfilingFilename)
	if err != nil {
		t.Error("No heap profile found on filesystem", err)
		return
	}
	defer file.Close()

	info, err = file.Stat()
	if err != nil {
		t.Error("Cannot get file information for mem profile", err)
		return
	}

	if info.Size() == 0 {
		t.Error("Empty heap profiling file", err)
		return
	}

	// remove profiling
	os.Remove(CPUProfilingFilename)
	os.Remove(MemProfilingFilename)

}
