//go:build tklnativeheap && cgo

package nativeheap

/*
#include <malloc/malloc.h>

static long long native_heap_in_use(void) {
	malloc_statistics_t stats;
	malloc_zone_statistics(NULL, &stats);
	return (long long)stats.size_in_use;
}
*/
import "C"

// InUse returns the bytes in use across all malloc zones.
func InUse() (int64, bool) { return int64(C.native_heap_in_use()), true }
