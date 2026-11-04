// Package goheap reports Go heap memory still held by allocations from a given
// function, for tests that assert memory is not retained. Callers must set
// runtime.MemProfileRate = 1 before the allocations they want to observe.
package goheap

import (
	"runtime"
	"strings"
)

// LiveBytesAllocatedBy returns the in-use bytes of allocations whose stack
// contains a function whose name ends with function.
func LiveBytesAllocatedBy(function string) int64 {
	runtime.GC()
	runtime.GC()

	n, _ := runtime.MemProfile(nil, false)
	records := make([]runtime.MemProfileRecord, n+64)
	n, _ = runtime.MemProfile(records, false)

	var live int64
	for _, r := range records[:n] {
		frames := runtime.CallersFrames(r.Stack())
		for {
			f, more := frames.Next()
			if strings.HasSuffix(f.Function, function) {
				live += r.InUseBytes()
				break
			}
			if !more {
				break
			}
		}
	}
	return live
}
