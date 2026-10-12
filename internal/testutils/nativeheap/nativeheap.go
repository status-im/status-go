//go:build !(tklnativeheap && cgo && (darwin || linux))

// Package nativeheap reports the C heap in use, for benchmarks that measure
// memory held by native libraries. It is only live when built with the
// tklnativeheap tag and cgo on darwin or linux.
package nativeheap

// InUse returns the bytes malloc holds in use, and false when no probe is built in.
func InUse() (int64, bool) { return 0, false }
