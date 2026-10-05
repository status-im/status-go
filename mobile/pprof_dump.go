package statusgo

import (
	"encoding/json"
	"os"
	"runtime"
	"runtime/pprof"
)

// WriteHeapProfile writes the Go heap profile (gzipped pprof proto) to path and the runtime
// MemStats to path+".memstats.json". For devices whose pprof port cannot be reached (iOS):
// the client writes into its own container and copies the files off.
func WriteHeapProfile(path string) string { return callWithResponse(writeHeapProfile, path) }

func writeHeapProfile(path string) string {
	return makeJSONResponse(writeHeapProfileFiles(path))
}

func writeHeapProfileFiles(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	b, err := json.Marshal(map[string]uint64{
		"Sys": ms.Sys, "HeapAlloc": ms.HeapAlloc, "HeapSys": ms.HeapSys, "HeapIdle": ms.HeapIdle,
		"HeapInuse": ms.HeapInuse, "HeapReleased": ms.HeapReleased, "NumGC": uint64(ms.NumGC),
		"Mallocs": ms.Mallocs, "TotalAlloc": ms.TotalAlloc,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path+".memstats.json", b, 0o600)
}
