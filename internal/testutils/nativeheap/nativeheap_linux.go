//go:build tklnativeheap && cgo

package nativeheap

/*
#include <malloc.h>

static long long native_heap_in_use(void) {
	struct mallinfo2 info = mallinfo2();
	return (long long)(info.uordblks + info.hblkhd);
}
*/
import "C"

// InUse returns the bytes glibc malloc holds in use, mmapped blocks included.
func InUse() (int64, bool) { return int64(C.native_heap_in_use()), true }
