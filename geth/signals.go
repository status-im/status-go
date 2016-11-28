package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent( const char *jsonEvent );
*/
import "C"

const (
	EventLocalStorageSet = "local_storage.set"
)

func SendSignal(data []byte) {
	C.StatusServiceSignalEvent(C.CString(string(data)))
}
