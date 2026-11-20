package statusgo

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteHeapProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heap.pb.gz")

	var r APIResponse
	require.NoError(t, json.Unmarshal([]byte(WriteHeapProfile(path)), &r))
	require.Empty(t, r.Error)

	profile, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(profile, []byte{0x1f, 0x8b}), "heap profile is not gzipped")

	raw, err := os.ReadFile(path + ".memstats.json")
	require.NoError(t, err)
	var stats map[string]uint64
	require.NoError(t, json.Unmarshal(raw, &stats))
	require.NotZero(t, stats["HeapSys"])
}

func TestWriteHeapProfileReportsUnwritablePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "heap.pb.gz")

	var r APIResponse
	require.NoError(t, json.Unmarshal([]byte(WriteHeapProfile(path)), &r))
	require.NotEmpty(t, r.Error)
}
