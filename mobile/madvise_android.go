//go:build android

package statusgo

import _ "unsafe" // for go:linkname

// The Go runtime only defaults to MADV_DONTNEED when GOOS == "linux" (runtime1.go); on android
// it keeps MADV_FREE, so scavenged heap stays resident until the kernel is under memory
// pressure and every memory meter keeps counting it. Flip the advice at init. The Android
// library is linked with -checklinkname=0; a Go upgrade that removes the variable fails to link.
//
//go:linkname runtimeAdviseUnused runtime.adviseUnused
var runtimeAdviseUnused uint32

const madvDontneed = 4 // MADV_DONTNEED on linux/android

func init() {
	runtimeAdviseUnused = madvDontneed
}
