package codex

/*
   #include <stdbool.h>
   #include <stdlib.h>
   #include "libcodex.h"

   typedef struct {
       int ret;
       char* msg;
       size_t len;
       uintptr_t h;
   } Resp;

   static void* allocResp(uintptr_t h) {
       Resp* r = (Resp*)calloc(1, sizeof(Resp));
       r->h = h;
       return r;
   }

   static void freeResp(void* resp) {
       if (resp != NULL) {
           free(resp);
       }
   }

   static int getRet(void* resp) {
       if (resp == NULL) {
           return 0;
       }
       Resp* m = (Resp*) resp;
       return m->ret;
   }
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime/cgo"
	"sync"
	"unsafe"
)

// bridgeCtx is used for managing the C-Go bridge calls.
// It contains a wait group for synchronizing the calls,
// a cgo.Handle for passing context to the C code,
// a response pointer for receiving data from the C code,
// and fields for storing the result and error of the call.
type bridgeCtx struct {
	wg     *sync.WaitGroup
	h      cgo.Handle
	resp   unsafe.Pointer
	result string
	err    error

	// Callback used for receiving progress updates during upload/download.
	//
	// For the upload, the bytes parameter indicates the number of bytes uploaded.
	// If the chunk size is superior or equal to the blocksize (passed in init function),
	// the callback will be called when a block is put in the store.
	// Otherwise, it will be called when a chunk is pushed into the stream.
	//
	// For the download, the bytes is the size of the chunk received, and the chunk
	// is the actual chunk of data received.
	onProgress func(bytes int, chunk []byte)
}

// newBridgeCtx creates a new bridge context for managing C-Go calls.
// The bridge context is initialized with a wait group and a cgo.Handle.
func newBridgeCtx() *bridgeCtx {
	bridge := &bridgeCtx{}
	bridge.wg = &sync.WaitGroup{}
	bridge.wg.Add(1)
	bridge.h = cgo.NewHandle(bridge)
	bridge.resp = C.allocResp(C.uintptr_t(uintptr(bridge.h)))
	return bridge
}

// callError creates an error message for a failed C-Go call.
func (b *bridgeCtx) callError(name string) error {
	return fmt.Errorf("failed the call to %s returned code %d", name, C.getRet(b.resp))
}

// free releases the resources associated with the bridge context,
// including the cgo.Handle and the response pointer.
func (b *bridgeCtx) free() {
	if b.h > 0 {
		b.h.Delete()
		b.h = 0
	}

	if b.resp != nil {
		C.freeResp(b.resp)
		b.resp = nil
	}
}

// callback is the function called by the C code to communicate back to Go.
// It handles progress updates, successful completions, and errors.
// The function uses the response pointer to retrieve the bridge context
// and update its state accordingly.
//
//export callback
func callback(ret C.int, msg *C.char, len C.size_t, resp unsafe.Pointer) {
	if resp == nil {
		return
	}

	m := (*C.Resp)(resp)
	m.ret = ret
	m.msg = msg
	m.len = len

	if m.h == 0 {
		return
	}

	h := cgo.Handle(m.h)
	if h == 0 {
		return
	}

	if v, ok := h.Value().(*bridgeCtx); ok {
		switch ret {
		case C.RET_PROGRESS:
			if v.onProgress == nil {
				return
			}
			if msg != nil {
				chunk := C.GoBytes(unsafe.Pointer(msg), C.int(len))
				v.onProgress(int(C.int(len)), chunk)
			} else {
				v.onProgress(int(C.int(len)), nil)
			}
		case C.RET_OK:
			retMsg := C.GoStringN(msg, C.int(len))
			v.result = retMsg
			v.err = nil
			if v.wg != nil {
				v.wg.Done()
			}
		case C.RET_ERR:
			retMsg := C.GoStringN(msg, C.int(len))
			v.err = errors.New(retMsg)
			if v.wg != nil {
				v.wg.Done()
			}
		}
	}
}

// wait waits for the bridge context to complete its operation.
// It returns the result and error of the operation.
func (b *bridgeCtx) wait() (string, error) {
	b.wg.Wait()
	return b.result, b.err
}
