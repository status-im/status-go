//go:build darwin && cgo
// +build darwin,cgo

package library

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#include <stddef.h>
#include <stdbool.h>
extern bool ServiceSignalEvent( const char *jsonEvent );
*/
import "C"
