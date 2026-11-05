//go:build android && cgo

package backend

/*
#include <malloc.h>

#ifndef M_PURGE
#define M_PURGE (-101)
#endif
#ifndef M_PURGE_ALL
#define M_PURGE_ALL (-104)
#endif

// M_PURGE_ALL (API 34) also drains scudo's caches; older bionic rejects it with 0.
static void status_purge_native_heap(void) {
	if (mallopt(M_PURGE_ALL, 0) == 0) {
		mallopt(M_PURGE, 0);
	}
}
*/
import "C"

// purgeNativeHeap returns the native allocator's freed pages to the OS. Bionic only does this
// on its own when allocation activity drives decay, so an idle foreground process keeps them.
func purgeNativeHeap() {
	C.status_purge_native_heap()
}
