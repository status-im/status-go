package profiling

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProfilingCPU(t *testing.T) {
	dir := t.TempDir()

	err := StartCPUProfile(dir)
	require.NoError(t, err)

	// Block for a bit to collect some metrics.
	time.Sleep(time.Second)

	err = StopCPUProfile()
	require.NoError(t, err)

	// Verify that the file has some content.
	file, err := os.Open(filepath.Join(dir, CPUFilename))
	require.NoError(t, err)
	defer func() {
		err := file.Close()
		require.NoError(t, err)
	}()

	t.Logf("CPU profile saved in %s for %s", filepath.Join(dir, CPUFilename), os.Args[0])

	info, err := file.Stat()
	require.NoError(t, err)
	require.True(t, info.Size() > 0, "a file with CPU profile is empty")
}

func TestProfilingMem(t *testing.T) {
	dir := t.TempDir()

	err := WriteHeapFile(dir)
	require.NoError(t, err)

	// Verify that the file has some content.
	file, err := os.Open(filepath.Join(dir, MemFilename))
	require.NoError(t, err)
	defer func() {
		err := file.Close()
		require.NoError(t, err)
	}()

	t.Logf("Memory profile saved in %s for %s", filepath.Join(dir, MemFilename), os.Args[0])

	info, err := file.Stat()
	require.NoError(t, err)
	require.True(t, info.Size() > 0, "a file with memory profile is empty")
}
