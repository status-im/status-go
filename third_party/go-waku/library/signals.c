// ======================================================================================
// cgo compilation (for desktop platforms and local tests)
// ======================================================================================

#include <stdio.h>
#include <stddef.h>
#include <stdbool.h>
#include "_cgo_export.h"

typedef void (*callback)(int retCode, const char *jsonEvent, void* userData);

bool ServiceSignalEvent(void *cb, const char *jsonEvent) {
	if (cb) {
		((callback)cb)(0, jsonEvent, NULL);
	}

	return true;
}
