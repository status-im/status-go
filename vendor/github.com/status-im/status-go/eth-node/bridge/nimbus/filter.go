// +build nimbus

package nimbusbridge

// https://golang.org/cmd/cgo/

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <libnimbus.h>
*/
import "C"

import (
	"unsafe"

	"github.com/status-im/status-go/eth-node/types"
)

type nimbusFilterWrapper struct {
	filter *C.filter_options
	id     string
	own    bool
}

// NewNimbusFilterWrapper returns an object that wraps Nimbus's Filter in a types interface
func NewNimbusFilterWrapper(f *C.filter_options, id string, own bool) types.Filter {
	wrapper := &nimbusFilterWrapper{
		filter: f,
		id:     id,
		own:    own,
	}
	return wrapper
}

// GetNimbusFilterFrom retrieves the underlying whisper Filter struct from a wrapped Filter interface
func GetNimbusFilterFrom(f types.Filter) *C.filter_options {
	return f.(*nimbusFilterWrapper).filter
}

// ID returns the filter ID
func (w *nimbusFilterWrapper) ID() string {
	return w.id
}

// Free frees the C memory associated with the filter
func (w *nimbusFilterWrapper) Free() {
	if !w.own {
		panic("native filter is not owned by Go")
	}

	if w.filter.privateKeyID != nil {
		C.free(unsafe.Pointer(w.filter.privateKeyID))
		w.filter.privateKeyID = nil
	}
	if w.filter.symKeyID != nil {
		C.free(unsafe.Pointer(w.filter.symKeyID))
		w.filter.symKeyID = nil
	}
}
