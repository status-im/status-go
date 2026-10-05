//go:build android

package statusgo

import (
	"sync/atomic"
	_ "unsafe" // for go:linkname
)

// The Go runtime only defaults to MADV_DONTNEED when GOOS == "linux" (runtime1.go); on android
// it keeps MADV_FREE, so scavenged heap stays resident until the kernel is under memory
// pressure and every memory meter keeps counting it. GODEBUG=madvdontneed=1 is not accepted
// by //go:debug or go.mod, and the env is fixed before the library loads, so flip
// runtime.adviseUnused (mem_linux.go sysUnusedOS) instead. The library is linked with
// -checklinkname=0, so a renamed runtime variable would not fail the link: the toolchain
// version is pinned in madvise_android_unverified.go.
//
//go:linkname runtimeAdviseUnused runtime.adviseUnused
var runtimeAdviseUnused uint32

const (
	madvFree     = 8 // MADV_FREE on linux/android
	madvDontneed = 4 // MADV_DONTNEED on linux/android
)

func init() {
	// The scavenger may already be running; only replace the default so a runtime fallback
	// (MADV_FREE unsupported) is kept.
	atomic.CompareAndSwapUint32(&runtimeAdviseUnused, madvFree, madvDontneed)
}
